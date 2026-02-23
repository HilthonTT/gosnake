package views

import (
	"errors"

	"github.com/Broderick-Westrope/charmutils"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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

	formBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FF41")).
			Padding(1, 3)

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
					Validate(func(s string) error {
						if len(s) == 0 {
							return errors.New("username cannot be empty")
						}
						return nil
					}),
				huh.NewSelect[int]().
					Value(&formData.Level).
					Title("Starting Level").
					Description("Higher levels start faster").
					Options(charmutils.HuhIntRangeOptions(1, 10)...),
			),
		).
			WithKeyMap(keys.formKeys).
			WithTheme(greenTheme()),
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
	case tea.KeyMsg:
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

func (m *MenuModel) View() string {
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

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *MenuModel) announceCompletion() tea.Cmd {
	m.hasAnnouncedCompletion = true
	in := tui.NewSingleInput(tui.ModeGame, m.formData.Level, m.formData.Username)
	return tui.SwitchModeCmd(tui.ModeGame, in)
}

func greenTheme() *huh.Theme {
	t := huh.ThemeBase()
	t.Focused.Title = t.Focused.Title.Foreground(lipgloss.Color("#00FF41"))
	t.Blurred.Title = t.Blurred.Title.Foreground(lipgloss.Color("#006400"))
	return t
}
