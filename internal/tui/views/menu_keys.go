package views

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

var _ help.KeyMap = (*menuKeyMap)(nil)

type menuKeyMap struct {
	Exit        key.Binding
	Leaderboard key.Binding
	Next        key.Binding
	Prev        key.Binding
	Left        key.Binding
	Right       key.Binding
	Play        key.Binding
}

func defaultMenuKeyMap() *menuKeyMap {
	return &menuKeyMap{
		Exit:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "quit")),
		Leaderboard: key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "leaderboard")),
		Next:        key.NewBinding(key.WithKeys("tab", "down"), key.WithHelp("↓/tab", "next")),
		Prev:        key.NewBinding(key.WithKeys("shift+tab", "up"), key.WithHelp("↑", "back")),
		Left:        key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←", "change")),
		Right:       key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("←/→", "change")),
		Play:        key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
	}
}

func (k *menuKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Next, k.Right, k.Play, k.Leaderboard, k.Exit}
}

func (k *menuKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{k.ShortHelp()}
}
