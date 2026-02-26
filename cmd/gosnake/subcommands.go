package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/starter"
	"github.com/HilthonTT/gosnake/server"
	tea "github.com/charmbracelet/bubbletea"
)

type MenuCmd struct{}

func (c *MenuCmd) Run(globals *GlobalVars) error {
	return launchStarter(globals, tui.ModeMenu, tui.NewMenuInput())
}

type PlayCmd struct {
	Level int    `help:"Level to start at" short:"l" default:"1"`
	Name  string `help:"Name of the player" short:"n" default:"Anonymous"`
}

func (c *PlayCmd) Run(globals *GlobalVars) error {
	return launchStarter(globals, tui.ModeGame, tui.NewSingleInput(tui.ModeGame, c.Level, c.Name))
}

type LeaderboardCmd struct{}

func (c *LeaderboardCmd) Run(globals *GlobalVars) error {
	return launchStarter(globals, tui.ModeLeaderboard, tui.NewLeaderboardInput())
}

func launchStarter(globals *GlobalVars, starterMode tui.Mode, switchIn tui.SwitchModeInput) error {
	db, err := data.NewDB(globals.DB)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}

	cfg, err := config.GetConfig(globals.Config)
	if err != nil {
		return fmt.Errorf("getting config: %w", err)
	}

	model, err := starter.NewModel(
		starter.NewInput(starterMode, db, cfg, switchIn),
	)
	if err != nil {
		return fmt.Errorf("creating starter model: %w", err)
	}

	exitModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return fmt.Errorf("failed to run program: %w", err)
	}

	typedExitModel, ok := exitModel.(*starter.Model)
	if !ok {
		return fmt.Errorf("faield to assert exit model type: %w", err)
	}

	if err = typedExitModel.ExitError; err != nil {
		return fmt.Errorf("starter model exited with an error: %w", err)
	}

	return nil
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

	log.Printf("GoSnake multiplayer server starting on %s:%d", c.Host, c.Port)
	log.Printf("Players connect with: ssh <name>@<host> -p %d -t <room-id>", c.Port)

	// Start serving in the background.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	// Block until SIGINT / SIGTERM or a fatal server error.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("Received %s — shutting down...", sig)
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Println("Server stopped.")
	return nil
}
