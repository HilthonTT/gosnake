package views

import "github.com/charmbracelet/bubbles/key"

type leaderboardKeyMap struct {
	Exit      key.Binding
	Help      key.Binding
	Left      key.Binding
	Right     key.Binding
	Up        key.Binding
	Down      key.Binding
	Search    key.Binding
	SortScore key.Binding
	SortLevel key.Binding
	SortName  key.Binding
	SortMode  key.Binding
	SortDate  key.Binding
}

func defaultLeaderboardKeyMap() *leaderboardKeyMap {
	return &leaderboardKeyMap{
		Exit:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("escape", "exit")),
		Help:  key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Left:  key.NewBinding(key.WithKeys("left"), key.WithHelp("left arrow", "move left")),
		Right: key.NewBinding(key.WithKeys("right"), key.WithHelp("right arrow", "move right")),
		Up:    key.NewBinding(key.WithKeys("up"), key.WithHelp("up arrow", "move up")),
		Down:  key.NewBinding(key.WithKeys("down"), key.WithHelp("down arrow", "move down")),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		SortScore: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort by score"),
		),
		SortLevel: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "sort by level"),
		),
		SortName: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "sort by name"),
		),
		SortMode: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "sort by mode"),
		),
		SortDate: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "sort by date"),
		),
	}
}

func (k *leaderboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Exit,
		k.Help,
	}
}

func (k *leaderboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			k.Exit,
			k.Help,
		},
		{
			k.Left,
			k.Right,
		},
		{
			k.Up,
			k.Down,
		},
		{
			k.Search,
			k.Search,
		},
		{k.SortScore, k.SortLevel, k.SortName},
		{k.SortMode, k.SortDate},
	}
}
