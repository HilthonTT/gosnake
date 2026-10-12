package views

import "charm.land/bubbles/v2/key"

type leaderboardKeyMap struct {
	Exit      key.Binding
	Back      key.Binding
	Help      key.Binding
	Replay    key.Binding
	Up        key.Binding
	Down      key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Top       key.Binding
	Bottom    key.Binding
	Search    key.Binding
	NextMode  key.Binding
	PrevMode  key.Binding
	SortScore key.Binding
	SortLevel key.Binding
	SortName  key.Binding
	SortMode  key.Binding
	SortDate  key.Binding

	// Sort is display-only: it summarises the sort keys in the short help.
	Sort key.Binding
}

func defaultLeaderboardKeyMap() *leaderboardKeyMap {
	return &leaderboardKeyMap{
		Exit:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "menu")),
		Back:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "menu")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
		Replay:    key.NewBinding(key.WithKeys("enter", "r"), key.WithHelp("enter", "play again")),
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		PageUp:    key.NewBinding(key.WithKeys("pgup", "ctrl+u"), key.WithHelp("pgup", "page up")),
		PageDown:  key.NewBinding(key.WithKeys("pgdown", "ctrl+d"), key.WithHelp("pgdn", "page down")),
		Top:       key.NewBinding(key.WithKeys("home", "g"), key.WithHelp("g/home", "top")),
		Bottom:    key.NewBinding(key.WithKeys("end", "G"), key.WithHelp("G/end", "bottom")),
		Search:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		NextMode:  key.NewBinding(key.WithKeys("tab", "right"), key.WithHelp("tab", "mode")),
		PrevMode:  key.NewBinding(key.WithKeys("shift+tab", "left"), key.WithHelp("shift+tab", "prev mode")),
		SortScore: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort by score")),
		SortLevel: key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "sort by level")),
		SortName:  key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "sort by name")),
		SortMode:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "sort by mode")),
		SortDate:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "sort by date")),
		Sort:      key.NewBinding(key.WithKeys("s", "l", "n", "m", "d"), key.WithHelp("s/l/n/m/d", "sort")),
	}
}

func (k *leaderboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.NextMode, k.Search, k.Replay, k.Exit, k.Help}
}

func (k *leaderboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.Top, k.Bottom, k.NextMode, k.PrevMode},
		{k.SortScore, k.SortLevel, k.SortName, k.SortMode, k.SortDate},
		{k.Search, k.Replay, k.Exit, k.Help},
	}
}
