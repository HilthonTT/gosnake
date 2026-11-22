package views

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/theme"
	"github.com/HilthonTT/gosnake/pkg/snake"
)

const (
	// maxUsernameLen keeps names short enough for the sidebar and leaderboard.
	maxUsernameLen = 20

	// menuLargeLogoHeight is the terminal height below which the menu swaps
	// the pixel logo for a one-line wordmark.
	menuLargeLogoHeight = 30

	tileGap  = 2
	maxTileW = 20
	minTileW = 14
)

// menuField is a focusable row of the menu.
type menuField int

const (
	fieldName menuField = iota
	fieldMode
	fieldLevel
	fieldPlay
	fieldCount
)

type modeTile struct {
	mode  tui.Mode
	title string
	desc  string
	color lipgloss.Style
}

var modeTiles = []modeTile{
	{tui.ModeNormal, "Normal", "Classic snake. Eat and grow.", lipgloss.NewStyle().Foreground(theme.Accent)},
	{tui.ModeCrazy, "Crazy", "Bombs blink, then turn deadly.", lipgloss.NewStyle().Foreground(theme.Danger)},
	{tui.ModeAI, "Versus AI", "Share the board with a hungry bot.", lipgloss.NewStyle().Foreground(theme.Rival)},
}

var _ tea.Model = &MenuModel{}

// MenuModel is the start screen: player name, mode and starting level.
type MenuModel struct {
	keys *menuKeyMap
	help help.Model

	name    textinput.Model
	mode    int // index into modeTiles
	level   int
	focus   menuField
	nameErr string
	started bool

	width  int
	height int
}

func NewMenuModel(in *tui.MenuInput) *MenuModel {
	name := textinput.New()
	name.Placeholder = "your name"
	name.CharLimit = maxUsernameLen
	name.Prompt = "› "
	name.SetWidth(maxUsernameLen + 1)
	styles := textinput.DefaultStyles(true)
	styles.Focused.Prompt = theme.Key
	styles.Focused.Placeholder = theme.Hint
	styles.Blurred.Prompt = theme.Hint
	styles.Blurred.Placeholder = theme.Hint
	styles.Cursor.Color = theme.Accent
	name.SetStyles(styles)

	m := &MenuModel{
		keys:  defaultMenuKeyMap(),
		help:  theme.NewHelp(),
		name:  name,
		level: 1,
	}

	if in != nil {
		m.name.SetValue(in.Username)
		for i, t := range modeTiles {
			if t.mode == in.Mode {
				m.mode = i
			}
		}
		if in.Level >= 1 && in.Level <= snake.MaxLevel {
			m.level = in.Level
		}
	}

	// Returning players land on the Play button, ready to go again.
	if strings.TrimSpace(m.name.Value()) != "" {
		m.focus = fieldPlay
	}
	return m
}

func (m *MenuModel) Init() tea.Cmd {
	return m.setFocus(m.focus)
}

func (m *MenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		return m, m.keyUpdate(msg)
	}

	var cmd tea.Cmd
	m.name, cmd = m.name.Update(msg)
	return m, cmd
}

func (m *MenuModel) keyUpdate(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Exit):
		return tea.Quit
	case key.Matches(msg, m.keys.Leaderboard):
		return tui.SwitchModeCmd(tui.ModeLeaderboard, tui.NewLeaderboardInput())
	case key.Matches(msg, m.keys.Next):
		return m.setFocus((m.focus + 1) % fieldCount)
	case key.Matches(msg, m.keys.Prev):
		return m.setFocus((m.focus + fieldCount - 1) % fieldCount)
	case key.Matches(msg, m.keys.Play):
		if m.focus == fieldName && m.validName() {
			return m.setFocus(fieldMode)
		}
		return m.start()
	}

	if m.focus == fieldName {
		var cmd tea.Cmd
		m.name, cmd = m.name.Update(msg)
		if m.nameErr != "" && strings.TrimSpace(m.name.Value()) != "" {
			m.nameErr = ""
		}
		return cmd
	}

	delta := 0
	switch {
	case key.Matches(msg, m.keys.Left):
		delta = -1
	case key.Matches(msg, m.keys.Right):
		delta = 1
	}

	switch m.focus {
	case fieldMode:
		m.mode = (m.mode + delta + len(modeTiles)) % len(modeTiles)
	case fieldLevel:
		m.level = min(max(m.level+delta, 1), snake.MaxLevel)
		if s := msg.String(); len(s) == 1 && s[0] >= '0' && s[0] <= '9' {
			m.level = int(s[0] - '0')
			if m.level == 0 {
				m.level = 10
			}
			m.level = min(m.level, snake.MaxLevel)
		}
	}
	return nil
}

// setFocus moves focus to f, focusing or blurring the name input to match.
func (m *MenuModel) setFocus(f menuField) tea.Cmd {
	m.focus = f
	if f == fieldName {
		return m.name.Focus()
	}
	m.name.Blur()
	return nil
}

func (m *MenuModel) validName() bool {
	if strings.TrimSpace(m.name.Value()) == "" {
		m.nameErr = "enter a name to play"
		return false
	}
	m.nameErr = ""
	return true
}

func (m *MenuModel) start() tea.Cmd {
	if m.started {
		return nil
	}
	if !m.validName() {
		return m.setFocus(fieldName)
	}
	m.started = true
	mode := modeTiles[m.mode].mode
	in := tui.NewSingleInput(mode, m.level, strings.TrimSpace(m.name.Value()))
	return tui.SwitchModeCmd(mode, in)
}

func (m *MenuModel) tileWidth() int {
	if m.width == 0 {
		return maxTileW
	}
	avail := (m.width - 4 - tileGap*(len(modeTiles)-1)) / len(modeTiles)
	return min(max(avail, minTileW), maxTileW)
}

func (m *MenuModel) View() tea.View {
	tileW := m.tileWidth()
	contentW := tileW*len(modeTiles) + tileGap*(len(modeTiles)-1)

	small := m.height > 0 && m.height < menuLargeLogoHeight ||
		m.width > 0 && m.width < theme.LogoWidth+4

	header := []string{theme.Logo(small)}
	if small {
		header = append(header, theme.Subtitle.Render("eat · grow · survive"))
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		m.nameView(contentW),
		"",
		m.modeView(contentW, tileW),
		"",
		m.levelView(contentW),
		"",
		lipgloss.PlaceHorizontal(contentW, lipgloss.Center, m.playView()),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.JoinVertical(lipgloss.Center, header...),
		"",
		body,
		"",
		m.help.View(m.keys),
	)
	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

// label renders a section heading, lit when its field has focus.
func (m *MenuModel) label(f menuField, title string, width int) string {
	c := theme.Subtle
	if m.focus == f {
		c = theme.Accent
	}
	return theme.Section(title, width, c)
}

func (m *MenuModel) nameView(w int) string {
	errLine := ""
	if m.nameErr != "" {
		errLine = theme.Error.Render("  " + m.nameErr)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.label(fieldName, "Player", w), m.name.View(), errLine)
}

func (m *MenuModel) modeView(w, tileW int) string {
	focused := m.focus == fieldMode
	tiles := make([]string, len(modeTiles))
	for i, t := range modeTiles {
		selected := i == m.mode

		title := lipgloss.NewStyle().Bold(true).Width(tileW).Padding(0, 1)
		desc := lipgloss.NewStyle().Width(tileW).Padding(0, 1).Height(2).MaxHeight(2)
		rule := lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", tileW))
		marker := "  "

		switch {
		case selected && focused:
			title = title.Foreground(theme.OnAccent).Background(t.color.GetForeground())
			desc = desc.Foreground(theme.Muted)
			rule = t.color.Render(strings.Repeat("━", tileW))
		case selected:
			title = title.Inherit(t.color)
			desc = desc.Foreground(theme.Muted)
			rule = t.color.Render(strings.Repeat("━", tileW))
		default:
			title = title.Foreground(theme.Subtle)
			desc = desc.Foreground(theme.Border)
		}
		if selected {
			marker = "● "
		}

		tiles[i] = lipgloss.JoinVertical(lipgloss.Left,
			title.Render(marker+t.title),
			desc.Render(t.desc),
			rule,
		)
	}

	gap := strings.Repeat(" ", tileGap)
	row := tiles[0]
	for _, t := range tiles[1:] {
		row = lipgloss.JoinHorizontal(lipgloss.Top, row, gap, t)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.label(fieldMode, "Mode", w), row)
}

func (m *MenuModel) levelView(w int) string {
	arrow := theme.Hint
	if m.focus == fieldLevel {
		arrow = theme.Key
	}
	left, right := arrow.Render("◂"), arrow.Render("▸")
	if m.level == 1 {
		left = theme.Hint.Faint(true).Render("◂")
	}
	if m.level == snake.MaxLevel {
		right = theme.Hint.Faint(true).Render("▸")
	}

	meter := left + " " + theme.Meter(snake.MaxLevel, m.level) + " " + right
	number := theme.Value.Render(fmt.Sprintf("%2d", m.level)) + theme.Hint.Render(fmt.Sprintf("/%d", snake.MaxLevel))

	return lipgloss.JoinVertical(lipgloss.Left,
		m.label(fieldLevel, "Starting level", w),
		theme.SpaceBetween(w, meter, number),
		theme.Hint.Render(levelPace(m.level)),
	)
}

func (m *MenuModel) playView() string {
	style := lipgloss.NewStyle().Bold(true).Padding(0, 3)
	if m.focus == fieldPlay {
		style = style.Foreground(theme.OnAccent).Background(theme.Accent)
	} else {
		style = style.Foreground(theme.Muted).Background(lipgloss.Color("#27272A"))
	}
	return style.Render("▶  PLAY")
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
