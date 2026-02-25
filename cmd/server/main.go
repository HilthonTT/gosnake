package main

import (
	"github.com/adrg/xdg"
	"github.com/alecthomas/kong"
)

type CLI struct {
	GlobalVars

	Menu        MenuCmd        `cmd:"" help:"Start in the menu" default:"1"`
	Play        PlayCmd        `cmd:"" help:"Start in the game directly"`
	Leaderboard LeaderboardCmd `cmd:"" help:"Start on the leaderboard"`
	Serve       ServeCmd       `cmd:"" help:"Start a multiplayer SSH server"`
}

type GlobalVars struct {
	Config string `help:"Path to config file. Empty value will use XDG config directory." default:""`
	DB     string `help:"Path to database file. Empty value will use XDG data directory." default:""`
}

func main() {
	cli := CLI{}
	ctx := kong.Parse(&cli,
		kong.Name("gosnake"),
		kong.Description("A snake TUI written in Go — singleplayer and multiplayer over SSH"),
		kong.UsageOnError(),
	)

	if err := handleDefaultGlobals(&cli.GlobalVars); err != nil {
		ctx.FatalIfErrorf(err)
	}

	err := ctx.Run(&cli.GlobalVars)
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
		g.DB, err = xdg.DataFile("./gosnake/gosnake.db")
		if err != nil {
			return err
		}
	}
	return nil
}
