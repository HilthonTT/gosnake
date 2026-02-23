package components

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

var _ help.KeyMap = (*GameKeyMap)(nil)

// GameKeyMap holds all key bindings for the snake game.
type GameKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Pause     key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
	Help      key.Binding
}

// NewGameKeyMap returns the default key map.
func NewGameKeyMap() *GameKeyMap {
	return &GameKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "w", "k"),
			key.WithHelp("↑/w", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "s", "j"),
			key.WithHelp("↓/s", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "a", "h"),
			key.WithHelp("←/a", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "d", "l"),
			key.WithHelp("→/d", "right"),
		),
		Pause: key.NewBinding(
			key.WithKeys("p", "esc"),
			key.WithHelp("p", "pause"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "force quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// ShortHelp satisfies help.KeyMap.
func (k *GameKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Pause, k.Quit, k.Help}
}

// FullHelp satisfies help.KeyMap.
func (k *GameKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Pause, k.Quit, k.ForceQuit, k.Help},
	}
}
