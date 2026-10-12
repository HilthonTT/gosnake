package components

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"

	"github.com/HilthonTT/gosnake/internal/config"
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

// NewGameKeyMap builds the game key map from the user's configured keys.
// A nil keys uses the defaults.
func NewGameKeyMap(keys *config.Keys) *GameKeyMap {
	if keys == nil {
		keys = config.DefaultKeys()
	}

	return &GameKeyMap{
		Up:        NewBinding(keys.Up, "up"),
		Down:      NewBinding(keys.Down, "down"),
		Left:      NewBinding(keys.Left, "left"),
		Right:     NewBinding(keys.Right, "right"),
		Pause:     NewBinding(keys.Pause, "pause"),
		Quit:      NewBinding(keys.Quit, "quit"),
		ForceQuit: NewBinding(keys.ForceQuit, "force quit"),
		Help:      NewBinding(keys.Help, "help"),
	}
}

// NewBinding creates a binding whose help text lists every key.
func NewBinding(keys []string, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(KeyLabel(keys), desc))
}

// KeyLabel renders key names for help text, e.g. ["up","w"] -> "↑/w".
func KeyLabel(keys []string) string {
	labels := make([]string, len(keys))
	for i, k := range keys {
		switch k {
		case "up":
			labels[i] = "↑"
		case "down":
			labels[i] = "↓"
		case "left":
			labels[i] = "←"
		case "right":
			labels[i] = "→"
		case " ", "space":
			labels[i] = "space"
		default:
			labels[i] = k
		}
	}
	return strings.Join(labels, "/")
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
