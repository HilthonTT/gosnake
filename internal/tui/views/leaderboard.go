package views

import (
	"slices"
	"strconv"
	"strings"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sortColumn int

const (
	sortByScore sortColumn = iota
	sortByLevel
	sortByName
	sortByMode
	sortByDate
)

type sortState struct {
	col  sortColumn
	desc bool
}

var _ tea.Model = &LeaderboardModel{}

type LeaderboardModel struct {
	keys    *leaderboardKeyMap
	help    help.Model
	table   table.Model
	entries []data.LeaderboardEntry
	focusID int
	sort    sortState
	width   int
	height  int
	search  textinput.Model
}

func NewLeaderboardModel(in *tui.LeaderboardInput) *LeaderboardModel {
	focusID := 0
	if in.NewEntry != nil {
		focusID = in.NewEntry.ID
	}
	si := textinput.New()
	si.Placeholder = "search by name..."
	si.CharLimit = 100
	si.Prompt = "/ "
	si.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF41"))
	si.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("255"))
	si.ShowSuggestions = true
	si.SetSuggestions(uniqueNames(in.Entries))

	defaultSort := sortState{col: sortByScore, desc: true}

	m := &LeaderboardModel{
		keys:    defaultLeaderboardKeyMap(),
		help:    help.New(),
		entries: in.Entries,
		focusID: focusID,
		search:  si,
		sort:    defaultSort,
	}
	m.table = buildLeaderboardTable(m.filteredSorted(), focusID, 0)
	return m
}

func (m *LeaderboardModel) Init() tea.Cmd {
	return nil
}

func (m *LeaderboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.search.Focused() {
			switch {
			case key.Matches(msg, m.keys.Exit):
				m.search.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.search, cmd = m.search.Update(msg)
			cmds = append(cmds, cmd)
			m.rebuildTable()
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.Exit):
			return m, tui.SwitchModeCmd(tui.ModeMenu, tui.NewMenuInput())
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Search):
			cmds = append(cmds, m.search.Focus())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, m.keys.SortScore):
			m.toggleSort(sortByScore)
		case key.Matches(msg, m.keys.SortLevel):
			m.toggleSort(sortByLevel)
		case key.Matches(msg, m.keys.SortName):
			m.toggleSort(sortByName)
		case key.Matches(msg, m.keys.SortMode):
			m.toggleSort(sortByMode)
		case key.Matches(msg, m.keys.SortDate):
			m.toggleSort(sortByDate)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.rebuildTable()
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *LeaderboardModel) View() string {
	sortHint := m.sortHint()

	var searchView string
	if m.search.Focused() {
		searchView = m.search.View()
	} else {
		searchView = hintStyle.Render("press / to search  •  s=score  l=level  n=name  m=mode  d=date")
	}

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		sortHint,
		m.table.View(),
		"",
		searchView,
		m.help.View(m.keys),
	)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *LeaderboardModel) sortHint() string {
	names := map[sortColumn]string{
		sortByScore: "Score",
		sortByLevel: "Level",
		sortByName:  "Name",
		sortByMode:  "Mode",
		sortByDate:  "Date",
	}
	dir := "↓"
	if !m.sort.desc {
		dir = "↑"
	}

	str := "sorted by: " + names[m.sort.col] + " " + dir
	str += "\n"
	str += "\n"

	return hintStyle.Render(lipgloss.PlaceHorizontal(m.width, lipgloss.Center, str))
}

func buildLeaderboardTable(entries []data.LeaderboardEntry, focusID int, termWidth int) table.Model {
	nameWidth := 16
	for _, e := range entries {
		if len(e.Name) > nameWidth {
			nameWidth = len(e.Name)
		}
	}

	if termWidth > 0 {
		const otherCols = 10 + 8 + 10 + 14 // score + level + mode + date
		maxName := termWidth - otherCols
		if maxName < 0 {
			maxName = 0
		}
		if nameWidth > maxName {
			nameWidth = maxName
		}
	}

	cols := []table.Column{
		{Title: "Name", Width: nameWidth},
		{Title: "Score", Width: 10},
		{Title: "Level", Width: 8},
		{Title: "Mode", Width: 10},
		{Title: "Date", Width: 14},
	}

	focusIndex := 0
	rows := make([]table.Row, len(entries))

	for i, e := range entries {
		if e.ID == focusID {
			focusIndex = i
		}

		name := e.Name
		// Visually mark the player's own new entry so it stands out immediately.
		if e.ID == focusID {
			name = "▶ " + name
		}

		date := e.CreatedAt
		if len(date) >= 10 {
			date = date[:10]
		}

		mode := string(e.Mode)
		if mode == "" {
			mode = "n/a"
		}

		rows[i] = table.Row{
			name,
			strconv.Itoa(e.Score),
			strconv.Itoa(e.Level),
			mode,
			date,
		}
	}

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithStyles(s),
	)
	t.SetCursor(focusIndex)

	return t
}

func (m *LeaderboardModel) toggleSort(col sortColumn) {
	if m.sort.col == col {
		m.sort.desc = !m.sort.desc
	} else {
		m.sort.col = col
		m.sort.desc = true
	}

	m.rebuildTable()
}

func (m *LeaderboardModel) rebuildTable() {
	m.table = buildLeaderboardTable(m.filteredSorted(), m.focusID, m.width)
}

func (m *LeaderboardModel) filteredSorted() []data.LeaderboardEntry {
	needle := strings.ToLower(strings.TrimSpace(m.search.Value()))

	filtered := make([]data.LeaderboardEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if needle == "" || strings.Contains(strings.ToLower(e.Name), needle) {
			filtered = append(filtered, e)
		}
	}

	slices.SortFunc(filtered, func(a, b data.LeaderboardEntry) int {
		var cmp int

		switch m.sort.col {
		case sortByScore:
			cmp = a.Score - b.Score
		case sortByLevel:
			cmp = a.Level - b.Level
		case sortByName:
			cmp = strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case sortByMode:
			cmp = strings.Compare(string(a.Mode), string(b.Mode))
		case sortByDate:
			cmp = strings.Compare(a.CreatedAt, b.CreatedAt)
		}

		if m.sort.desc {
			return -cmp
		}

		return cmp
	})

	return filtered
}

func uniqueNames(entries []data.LeaderboardEntry) []string {
	seen := make(map[string]struct{}, len(entries))
	names := make([]string, len(entries))

	for _, e := range entries {
		if _, ok := seen[e.Name]; !ok {
			seen[e.Name] = struct{}{}
			names = append(names, e.Name)
		}
	}

	return names
}
