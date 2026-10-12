package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/server"
	"github.com/HilthonTT/gosnake/internal/services/leaderboard"
	"github.com/HilthonTT/gosnake/internal/telemetry"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/starter"
	"github.com/HilthonTT/gosnake/internal/tui/views"
)

type MenuCmd struct{}

func (c *MenuCmd) Run(globals *GlobalVars) error {
	return launchStarter(globals, tui.ModeMenu, tui.NewMenuInput())
}

type PlayCmd struct {
	GameMode string `arg:"" help:"Game mode to play (normal, crazy, ai)" default:"normal" enum:"normal,crazy,ai"`
	Level    int    `help:"Level to start at (1-10)" short:"l" default:"1"`
	Name     string `help:"Name of the player" short:"n" default:"Anonymous"`
}

func (c *PlayCmd) Run(globals *GlobalVars) error {
	playerModes := map[string]tui.Mode{
		"normal": tui.ModeNormal,
		"crazy":  tui.ModeCrazy,
		"ai":     tui.ModeAI,
	}

	mode, ok := playerModes[c.GameMode]
	if !ok {
		return fmt.Errorf("invalid game mode: %s", c.GameMode)
	}

	return launchStarter(globals, mode, tui.NewSingleInput(mode, c.Level, c.Name))
}

type LeaderboardCmd struct{}

func (c *LeaderboardCmd) Run(globals *GlobalVars) error {
	return launchStarter(globals, tui.ModeLeaderboard, tui.NewLeaderboardInput())
}

func launchStarter(globals *GlobalVars, starterMode tui.Mode, switchIn tui.SwitchModeInput) error {
	cfg, err := config.GetConfig(globals.Config)
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}

	db, err := data.NewDB(globals.DB)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	submitter, err := newScoreSubmitter(cfg.Leaderboard)
	if err != nil {
		return err
	}

	model, err := starter.NewModel(
		starter.NewInput(starterMode, data.NewLeaderboardRepository(db), cfg, switchIn, submitter),
	)
	if err != nil {
		return fmt.Errorf("creating starter model: %w", err)
	}

	exitModel, err := tea.NewProgram(model).Run()

	// The starter recovers panics from views itself so it can attach the game
	// state; report them once the terminal has been restored.
	if crash := model.Crash(); crash != nil {
		return reportCrash(model, crash)
	}
	if err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	typedExitModel, ok := exitModel.(*starter.Model)
	if !ok {
		return fmt.Errorf("unexpected exit model type %T", exitModel)
	}

	if err = typedExitModel.ExitError; err != nil {
		return fmt.Errorf("starter model exited with an error: %w", err)
	}

	return nil
}

// newScoreSubmitter returns the online leaderboard client, or nil when no API
// URL is configured.
func newScoreSubmitter(cfg config.Leaderboard) (views.ScoreSubmitter, error) {
	if cfg.APIURL == "" {
		return nil, nil
	}

	svc, err := leaderboard.NewLeaderboardService(leaderboard.Options{
		BaseURL:            cfg.APIURL,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	})
	if err != nil {
		return nil, fmt.Errorf("configuring online leaderboard: %w", err)
	}
	return svc, nil
}

func reportCrash(model *starter.Model, crash *starter.Crash) error {
	w, h := model.TermSize()
	report := telemetry.NewCrashReport(
		crash.Value,
		crash.Stack,
		model.Recorder(),
		w, h,
		model.CurrentMode().String(),
		model.GameSnapshot(),
	)

	slog.Error("gosnake crashed", "panic", crash.Value)

	path, writeErr := telemetry.WriteCrashReport(report)
	if writeErr != nil {
		fmt.Fprintf(os.Stderr, "gosnake crashed: %v\n", crash.Value)
		fmt.Fprintf(os.Stderr, "failed to write crash report: %v\n", writeErr)
	} else {
		fmt.Fprintf(os.Stderr, "gosnake crashed — report saved to %s\n", path)
	}

	return fmt.Errorf("panic: %v", crash.Value)
}

type ServeCmd struct {
	Key  string `help:"Path to SSH host key file" default:"gosnake_server" env:"GOSNAKE_KEY"`
	Host string `help:"Host address to bind to (empty = all interfaces)" default:"" env:"GOSNAKE_HOST"`
	Port int    `help:"TCP port to listen on" default:"2222" env:"GOSNAKE_PORT"`
}

func (c *ServeCmd) Run(_ *GlobalVars) error {
	srv, err := server.NewServer(c.Key, c.Host, c.Port)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	slog.Info("GoSnake multiplayer server starting", "host", c.Host, "port", c.Port)
	slog.Info(fmt.Sprintf("Players connect with: ssh <name>@<host> -p %d -t <room-id>", c.Port))

	// Start serving in the background.
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	// Block until SIGINT / SIGTERM or a fatal server error.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
	case err := <-errCh:
		if errors.Is(err, server.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("server stopped")
	return nil
}
