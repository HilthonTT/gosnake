package tui

import tea "github.com/charmbracelet/bubbletea"

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
	ModeGame
	ModeLeaderboard
)

var modeToStrMap = map[Mode]string{
	ModeMenu:        "Menu",
	ModeGame:        "Game",
	ModeLeaderboard: "Leaderboard",
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
