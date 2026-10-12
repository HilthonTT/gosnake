package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"

	"github.com/HilthonTT/gosnake/internal/telemetry"
)

type CLI struct {
	GlobalVars

	Version kong.VersionFlag `help:"Print the version and exit." short:"v"`

	Menu        MenuCmd        `cmd:"" help:"Start in the menu" default:"1"`
	Play        PlayCmd        `cmd:"" help:"Start in the game"`
	Leaderboard LeaderboardCmd `cmd:"" help:"Start on the leaderboard"`
	Serve       ServeCmd       `cmd:"" help:"Start a multiplayer SSH server"`
}

type GlobalVars struct {
	Config   string     `help:"Path to config file. Empty value will use XDG config directory." default:"" env:"GOSNAKE_CONFIG"`
	DB       string     `help:"Path to database file. Empty value will use XDG data directory." default:"" env:"GOSNAKE_DB"`
	LogLevel slog.Level `help:"Log level (DEBUG, INFO, WARN, ERROR)" default:"INFO" env:"GOSNAKE_LOG_LEVEL"`
	LogFile  string     `help:"Path to log file, or '-' for stderr. Empty value uses the XDG state directory (stderr for 'serve')." default:"" env:"GOSNAKE_LOG_FILE"`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("gosnake"),
		kong.Description("A snake TUI written in Go"),
		kong.UsageOnError(),
		kong.Vars{"version": telemetry.Version},
	)

	if err := handleDefaultGlobals(&cli.GlobalVars, ctx.Command()); err != nil {
		ctx.FatalIfErrorf(err)
	}

	closeLog, err := setupLogger(cli.LogFile, cli.LogLevel)
	ctx.FatalIfErrorf(err)
	defer closeLog()

	// Call the Run() method of the selected parsed command.
	err = ctx.Run(&cli.GlobalVars)
	ctx.FatalIfErrorf(err)
}

func handleDefaultGlobals(g *GlobalVars, command string) error {
	var err error

	if g.Config == "" {
		if g.Config, err = xdg.ConfigFile("gosnake/config.toml"); err != nil {
			return fmt.Errorf("resolving config path: %w", err)
		}
	}

	if g.DB == "" {
		if g.DB, err = xdg.DataFile("gosnake/gosnake.db"); err != nil {
			return fmt.Errorf("resolving database path: %w", err)
		}
		migrateLegacyDB(g.DB)
	}

	if g.LogFile == "" {
		if command == "serve" {
			g.LogFile = "-"
		} else if g.LogFile, err = xdg.StateFile("gosnake/gosnake.log"); err != nil {
			return fmt.Errorf("resolving log path: %w", err)
		}
	}

	return nil
}

// migrateLegacyDB renames the database from its old "tetrigo.db" name so
// existing scores survive the rename.
func migrateLegacyDB(newPath string) {
	legacy := filepath.Join(filepath.Dir(newPath), "tetrigo.db")
	if _, err := os.Stat(newPath); !errors.Is(err, fs.ErrNotExist) {
		return
	}
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	if err := os.Rename(legacy, newPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not migrate %s to %s: %v\n", legacy, newPath, err)
	}
}

// setupLogger installs the default slog logger. It returns a function that
// closes the log file.
func setupLogger(logPath string, level slog.Level) (func(), error) {
	var (
		w       io.Writer = os.Stderr
		closeFn           = func() {}
	)

	if logPath != "-" {
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600) //nolint:gosec // path comes from the user's own flag/env
		if err != nil {
			return nil, fmt.Errorf("opening log file: %w", err)
		}
		w = f
		closeFn = func() { _ = f.Close() }
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if os.Getenv("ENV") == "production" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	slog.SetDefault(slog.New(handler))

	return closeFn, nil
}
