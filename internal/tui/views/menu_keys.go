package views

import (
	"charm.land/bubbles/v2/key"
	"charm.land/huh/v2"
)

type menuKeyMap struct {
	Exit        key.Binding
	Leaderboard key.Binding
	formKeys    *huh.KeyMap
}

func defaultMenuKeyMap() *menuKeyMap {
	keys := &menuKeyMap{
		Exit: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "quit"),
		),
		Leaderboard: key.NewBinding(
			key.WithKeys("ctrl+l"),
			key.WithHelp("ctrl+l", "leaderboard"),
		),
		formKeys: huh.NewDefaultKeyMap(),
	}
	keys.formKeys.Quit.SetEnabled(false)
	return keys
}
