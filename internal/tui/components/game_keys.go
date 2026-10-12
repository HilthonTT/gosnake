package components

import (
	"slices"
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
	Submit    key.Binding

	// Move is display-only: it summarises the four direction keys in help.
	Move key.Binding
}

// NewGameKeyMap builds the game key map from the user's configured keys.
// A nil keys uses the defaults.
func NewGameKeyMap(keys *config.Keys) *GameKeyMap {
	if keys == nil {
		keys = config.DefaultKeys()
	}

	km := &GameKeyMap{
		Up:        NewBinding(keys.Up, "up"),
		Down:      NewBinding(keys.Down, "down"),
		Left:      NewBinding(keys.Left, "left"),
		Right:     NewBinding(keys.Right, "right"),
		Pause:     NewBinding(keys.Pause, "pause"),
		Quit:      NewBinding(keys.Quit, "quit"),
		ForceQuit: NewBinding(keys.ForceQuit, "force quit"),
		Help:      NewBinding(keys.Help, "help"),
		Submit:    NewBinding(keys.Submit, "continue"),
		Move:      NewBinding(slices.Concat(keys.Up, keys.Down, keys.Left, keys.Right), "move"),
	}
	km.Move.SetHelp(moveLabel(keys), "move")
	return km
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

// moveLabel shows the first key of each direction, e.g. "↑↓←→".
func moveLabel(keys *config.Keys) string {
	var sb strings.Builder
	for _, dir := range [][]string{keys.Up, keys.Down, keys.Left, keys.Right} {
		if len(dir) > 0 {
			sb.WriteString(KeyLabel(dir[:1]))
		}
	}
	return sb.String()
}

// ShortHelp satisfies help.KeyMap.
func (k *GameKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Move, k.Pause, k.Quit, k.Help}
}

// FullHelp satisfies help.KeyMap.
func (k *GameKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Pause, k.Submit, k.Quit, k.ForceQuit, k.Help},
	}
}
