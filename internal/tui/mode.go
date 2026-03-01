package tui

import (
	"github.com/HilthonTT/gosnake/internal/data"
	tea "github.com/charmbracelet/bubbletea"
)

type SwitchModeMsg struct {
	Target Mode
	Input  SwitchModeInput
}

type SwitchModeInput interface {
	isSwitchModeInput()
}

func SwitchModeCmd(target Mode, in SwitchModeInput) tea.Cmd {
	return func() tea.Msg {
		return SwitchModeMsg{
			Target: target,
			Input:  in,
		}
	}
}

type Mode int

const (
	ModeMenu = Mode(iota)
	ModeLeaderboard
	ModeNormal
	ModeCrazy
)

var modeToStrMap = map[Mode]string{
	ModeMenu:        "Menu",
	ModeLeaderboard: "Leaderboard",
	ModeNormal:      "Normal",
	ModeCrazy:       "Crazy",
}

func (m Mode) String() string {
	return modeToStrMap[m]
}

type MenuInput struct{}

func NewMenuInput() *MenuInput {
	return &MenuInput{}
}

func (in *MenuInput) isSwitchModeInput() {}

type SingleInput struct {
	Mode     Mode
	Level    int
	Username string
}

func NewSingleInput(mode Mode, level int, username string) *SingleInput {
	return &SingleInput{
		Mode:     mode,
		Level:    level,
		Username: username,
	}
}

func (in *SingleInput) isSwitchModeInput() {}

type LeaderboardInput struct {
	NewEntry *data.LeaderboardEntry
	Entries  []data.LeaderboardEntry
}

func NewLeaderboardInput(opts ...func(input *LeaderboardInput)) *LeaderboardInput {
	in := &LeaderboardInput{}

	for _, opt := range opts {
		opt(in)
	}

	return in
}

func (in *LeaderboardInput) isSwitchModeInput() {}

func WithNewEntry(entry *data.LeaderboardEntry) func(input *LeaderboardInput) {
	return func(input *LeaderboardInput) {
		input.NewEntry = entry
	}
}

func (i *LeaderboardInput) SetEntries(entries []data.LeaderboardEntry) {
	i.Entries = entries
}
