package main

import (
	"fmt"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/starter"
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
