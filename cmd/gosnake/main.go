package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"
)

type CLI struct {
	GlobalVars

	Menu        MenuCmd        `cmd:"" help:"Start in the menu" default:"1"`
	Play        PlayCmd        `cmd:"" help:"Start in the game"`
	Leaderboard LeaderboardCmd `cmd:"" help:"Start on the leaderboard"`
	Serve       ServeCmd       `cmd:"" help:"Start a multiplayer SSH server"`
}

type GlobalVars struct {
	Config   string     `help:"Path to config file. Empty value will use XDG data directory." default:""`
	DB       string     `help:"Path to database file. Empty value will use XDG data directory." default:""`
	LogLevel slog.Level `help:"Log level (DEBUG, INFO, WARN, ERROR)" default:"INFO" env:"GOSNAKE_LOG_LEVEL"`
	LogFile  string     `help:"Path to log file." default:"gosnake.log" env:"GOSNAKE_LOG_FILE"`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("gosnake"),
		kong.Description("A snake TUI written in Go"),
		kong.UsageOnError(),
	)

	if err := handleDefaultGlobals(&cli.GlobalVars); err != nil {
		ctx.FatalIfErrorf(err)
	}

	logFile, err := setupLogger(cli.GlobalVars.LogFile, cli.GlobalVars.LogLevel)
	if err != nil {
		ctx.FatalIfErrorf(err)
	}
	defer logFile.Close()

	// Call the Run() method of the selected parsed command.
	err = ctx.Run(&cli.GlobalVars)
	ctx.FatalIfErrorf(err)
}

func handleDefaultGlobals(g *GlobalVars) error {
	if g.Config == "" {
		var err error
		g.Config, err = xdg.ConfigFile("./gosnake/config.toml")
		if err != nil {
			return err
		}
	}
	if g.DB == "" {
		var err error
		g.DB, err = xdg.DataFile("./gosnake/tetrigo.db")
		if err != nil {
			return err
		}
	}
	return nil
}

func setupLogger(logPath string, level slog.Level) (*os.File, error) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if os.Getenv("ENV") == "production" {
		handler = slog.NewJSONHandler(f, opts)
	} else {
		handler = slog.NewTextHandler(f, &slog.HandlerOptions{})
	}

	slog.SetDefault(slog.New(handler))

	return f, nil
}
