package views

import (
	"strconv"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/validate"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

const TitleStr = `
 ██████╗  ██████╗ ███████╗███╗   ██╗ █████╗ ██╗  ██╗███████╗
██╔════╝ ██╔═══██╗██╔════╝████╗  ██║██╔══██╗██║ ██╔╝██╔════╝
██║  ███╗██║   ██║███████╗██╔██╗ ██║███████║█████╔╝ █████╗  
██║   ██║██║   ██║╚════██║██║╚██╗██║██╔══██║██╔═██╗ ██╔══╝  
╚██████╔╝╚██████╔╝███████║██║ ╚████║██║  ██║██║  ██╗███████╗
 ╚═════╝  ╚═════╝ ╚══════╝╚═╝  ╚═══╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝`

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF41")).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#006400")).
			Italic(true)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444")).
			Italic(true)
)

var _ tea.Model = &MenuModel{}

type MenuModel struct {
	form                   *huh.Form
	hasAnnouncedCompletion bool
	keys                   *menuKeyMap
	formData               *MenuFormData

	width  int
	height int
}

type MenuFormData struct {
	Username string
	GameMode tui.Mode
	Level    int
}

func NewMenuModel(_ *tui.MenuInput) *MenuModel {
	formData := new(MenuFormData)
	keys := defaultMenuKeyMap()

	return &MenuModel{
		formData: formData,
		form: huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Value(&formData.Username).
					Title("Username").
					Placeholder("enter your name...").
					CharLimit(100).
					Validate(validate.NotEmpty("username")),
				huh.NewSelect[tui.Mode]().
					Value(&formData.GameMode).
					Title("Game Mode:").
					Options(
						huh.NewOption("Normal", tui.ModeNormal),
						huh.NewOption("Crazy", tui.ModeCrazy),
						huh.NewOption("AI", tui.ModeAI),
					),
				huh.NewSelect[int]().
					Value(&formData.Level).
					Title("Starting Level").
					Description("Higher levels start faster").
					Options(levelOptions()...),
			),
		).
			WithKeyMap(keys.formKeys).
			WithTheme(greenTheme()).
			WithShowHelp(true),
		keys: keys,
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
		if key.Matches(msg, m.keys.Exit) {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		titleWidth := lipgloss.Width(TitleStr)
		formWidth := min(m.width/2, titleWidth)
		m.form = m.form.WithWidth(formWidth)
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
	title := titleStyle.Render(TitleStr)
	subtitle := subtitleStyle.Render("eat, grow, survive")
	hint := hintStyle.Render("ctrl+c / esc to quit")

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		title,
		subtitle,
		"",
		m.form.View(),
		"",
		hint,
	)

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

func (m *MenuModel) announceCompletion() tea.Cmd {
	m.hasAnnouncedCompletion = true
	in := tui.NewSingleInput(m.formData.GameMode, m.formData.Level, m.formData.Username)
	return tui.SwitchModeCmd(m.formData.GameMode, in)
}

func greenTheme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		t := huh.ThemeBase(isDark)
		t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("#00FF41"))
		t.Blurred.Title = t.Blurred.Title.Foreground(lipgloss.Color("#006400"))
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
