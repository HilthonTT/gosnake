package views

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/theme"
	"github.com/HilthonTT/gosnake/internal/tui/validate"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

const (
	menuFormWidth = 40

	// maxUsernameLen keeps names short enough for the sidebar and leaderboard.
	maxUsernameLen = 20

	// menuLargeLogoHeight is the terminal height below which the menu swaps
	// the big logo for a one-line wordmark.
	menuLargeLogoHeight = 28
)

var modeDescriptions = map[tui.Mode]string{
	tui.ModeNormal: "Classic snake. Eat, grow, speed up.",
	tui.ModeCrazy:  "Bombs appear on the board. Dodge them!",
	tui.ModeAI:     "Race an AI snake for the same food.",
}

var _ tea.Model = &MenuModel{}

type MenuModel struct {
	form                   *huh.Form
	hasAnnouncedCompletion bool
	keys                   *menuKeyMap
	formData               *MenuFormData
	help                   help.Model

	width  int
	height int
}

type MenuFormData struct {
	Username string
	GameMode tui.Mode
	Level    int
}

func NewMenuModel(in *tui.MenuInput) *MenuModel {
	formData := &MenuFormData{GameMode: tui.ModeNormal, Level: 1}
	if in != nil {
		formData.Username = in.Username
		if _, ok := modeDescriptions[in.Mode]; ok {
			formData.GameMode = in.Mode
		}
		if in.Level >= 1 && in.Level <= snake.MaxLevel {
			formData.Level = in.Level
		}
	}
	keys := defaultMenuKeyMap()

	return &MenuModel{
		formData: formData,
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Value(&formData.Username).
					Title("Player").
					Placeholder("your name").
					CharLimit(maxUsernameLen).
					Validate(func(s string) error {
						return validate.NotEmpty("name")(strings.TrimSpace(s))
					}),
				huh.NewSelect[tui.Mode]().
					Value(&formData.GameMode).
					Title("Mode").
					Inline(true).
					Options(
						huh.NewOption("Normal", tui.ModeNormal),
						huh.NewOption("Crazy", tui.ModeCrazy),
						huh.NewOption("Versus AI", tui.ModeAI),
					).
					DescriptionFunc(func() string {
						return modeDescriptions[formData.GameMode]
					}, &formData.GameMode),
				huh.NewSelect[int]().
					Value(&formData.Level).
					Title("Starting level").
					Inline(true).
					Options(levelOptions()...).
					DescriptionFunc(func() string {
						return levelPace(formData.Level)
					}, &formData.Level),
			),
		).
			WithKeyMap(keys.formKeys).
			WithTheme(formTheme()).
			WithShowHelp(false).
			WithWidth(menuFormWidth),
		keys: keys,
		help: theme.NewHelp(),
	}
}

func (m *MenuModel) Init() tea.Cmd {
	if m.form == nil {
		return nil
	}
	return m.form.Init()
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, m.keys.Exit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Leaderboard):
			return m, tui.SwitchModeCmd(tui.ModeLeaderboard, tui.NewLeaderboardInput())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	var cmds []tea.Cmd

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
		cmds = append(cmds, cmd)
	}

	if m.form.State == huh.StateCompleted && !m.hasAnnouncedCompletion {
		cmds = append(cmds, m.announceCompletion())
	}

	return m, tea.Batch(cmds...)
}

func (m *MenuModel) View() tea.View {
	small := m.height > 0 && m.height < menuLargeLogoHeight ||
		m.width > 0 && m.width < theme.LogoWidth+2

	header := []string{theme.Logo(small)}
	if !small {
		header = append(header, "")
	}
	header = append(header, theme.Subtitle.Render("eat · grow · survive"))

	keys := keyList(append(m.form.KeyBinds(), m.keys.Leaderboard, m.keys.Exit))

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, header...),
		"",
		theme.Card.Render(m.form.View()),
		"",
		m.help.View(keys),
	)

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

func (m *MenuModel) announceCompletion() tea.Cmd {
	m.hasAnnouncedCompletion = true
	m.formData.Username = strings.TrimSpace(m.formData.Username)
	in := tui.NewSingleInput(m.formData.GameMode, m.formData.Level, m.formData.Username)
	return tui.SwitchModeCmd(m.formData.GameMode, in)
}

func formTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)
		accent := lipgloss.NewStyle().Foreground(theme.Accent)

		// No side bar on the focused field: vertical lines render dashed in
		// some terminals. The accent title and arrows mark focus instead.
		t.Focused.Base = lipgloss.NewStyle()
		t.Focused.Card = t.Focused.Base
		t.Focused.Title = t.Focused.Title.Foreground(theme.Accent).Bold(true)
		t.Focused.Description = t.Focused.Description.Foreground(theme.Subtle)
		t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(theme.Danger)
		t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(theme.Danger)
		t.Focused.SelectSelector = t.Focused.SelectSelector.Inherit(accent)
		t.Focused.NextIndicator = t.Focused.NextIndicator.Inherit(accent)
		t.Focused.PrevIndicator = t.Focused.PrevIndicator.Inherit(accent)
		t.Focused.SelectedOption = t.Focused.SelectedOption.Bold(true)
		t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(theme.Accent)
		t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(theme.Subtle)
		t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(theme.Accent)

		t.Blurred = t.Focused
		t.Blurred.Title = t.Blurred.Title.Foreground(theme.Muted).Bold(false)
		t.Blurred.SelectedOption = t.Blurred.SelectedOption.Bold(false)
		t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(theme.Subtle)
		t.Blurred.NextIndicator = lipgloss.NewStyle()
		t.Blurred.PrevIndicator = lipgloss.NewStyle()

		t.Help = theme.NewHelp().Styles
		return t
	})
}

// levelOptions lists every selectable starting level.
func levelOptions() []huh.Option[int] {
	opts := make([]huh.Option[int], 0, snake.MaxLevel)
	for level := 1; level <= snake.MaxLevel; level++ {
		opts = append(opts, huh.NewOption(strconv.Itoa(level), level))
	}
	return opts
}

// levelPace describes how fast a starting level feels.
func levelPace(level int) string {
	pace := "relaxed"
	switch {
	case level >= snake.MaxLevel:
		pace = "insane"
	case level >= 7:
		pace = "fast"
	case level >= 4:
		pace = "brisk"
	}
	tick := snake.GetTickInterval(level)
	return fmt.Sprintf("%s · one step every %dms", pace, tick.Milliseconds())
}
