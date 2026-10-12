package config

// Keys holds every configurable key binding. Each entry is a list of key names
// as understood by Bubble Tea (e.g. "up", "w", "ctrl+c", "space").
type Keys struct {
	ForceQuit []string `toml:"force_quit"`
	Exit      []string `toml:"exit"`
	Help      []string `toml:"help"`
	Submit    []string `toml:"submit"`
	Up        []string `toml:"up"`
	Down      []string `toml:"down"`
	Left      []string `toml:"left"`
	Right     []string `toml:"right"`
	Pause     []string `toml:"pause"`
	Quit      []string `toml:"quit"`
}

func DefaultKeys() *Keys {
	return &Keys{
		ForceQuit: []string{"ctrl+c"},
		Exit:      []string{"esc"},
		Help:      []string{"?"},
		Submit:    []string{"space", "enter"},
		Up:        []string{"up", "w", "k"},
		Down:      []string{"down", "s", "j"},
		Left:      []string{"left", "a", "h"},
		Right:     []string{"right", "d", "l"},
		Pause:     []string{"p", "esc"},
		Quit:      []string{"q"},
	}
}
