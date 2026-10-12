package views

import (
	"cmp"
	"fmt"
	"image/color"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/theme"
)

type sortColumn int

const (
	sortByScore sortColumn = iota
	sortByLevel
	sortByName
	sortByMode
	sortByDate
)

var sortColumnNames = map[sortColumn]string{
	sortByScore: "score",
	sortByLevel: "level",
	sortByName:  "name",
	sortByMode:  "mode",
	sortByDate:  "date",
}

type sortState struct {
	col  sortColumn
	desc bool
}

// modeTabs are the mode filters; the empty mode shows every entry.
var modeTabs = []struct {
	label string
	mode  data.GameMode
}{
	{"All", ""},
	{"Normal", data.GameModeNormal},
	{"Crazy", data.GameModeCrazy},
	{"Versus AI", data.GameModeAI},
}

// Column widths; the name column takes whatever is left.
const (
	colRankW  = 4
	colScoreW = 7
	colLevelW = 5
	colModeW  = 9
	colDateW  = 10
	colGap    = 2

	minNameW   = 10
	maxTableW  = 72
	minVisible = 3
)

var _ tea.Model = &LeaderboardModel{}

type LeaderboardModel struct {
	keys *leaderboardKeyMap
	help help.Model

	entries []data.LeaderboardEntry
	rows    []data.LeaderboardEntry // entries after the mode filter, search and sort
	ranks   map[int]int             // entry ID → score rank within the mode filter

	newEntry *data.LeaderboardEntry
	replay   *tui.SingleInput

	tab    int
	sort   sortState
	cursor int
	offset int
	search textinput.Model

	width  int
	height int
}

func NewLeaderboardModel(in *tui.LeaderboardInput) *LeaderboardModel {
	si := textinput.New()
	si.Placeholder = "search by name"
	si.CharLimit = maxUsernameLen
	si.Prompt = "/ "
	styles := textinput.DefaultStyles(true)
	styles.Focused.Prompt = theme.Key
	styles.Focused.Placeholder = theme.Hint
	styles.Blurred.Prompt = theme.Hint
	si.SetStyles(styles)
	si.ShowSuggestions = true
	si.SetSuggestions(uniqueNames(in.Entries))

	m := &LeaderboardModel{
		keys:     defaultLeaderboardKeyMap(),
		help:     theme.NewHelp(),
		entries:  in.Entries,
		newEntry: in.NewEntry,
		replay:   in.Replay,
		search:   si,
		sort:     sortState{col: sortByScore, desc: true},
	}
	m.keys.Replay.SetEnabled(in.Replay != nil)

	// Show the tab for the mode just played, since scores only compare
	// meaningfully within a mode.
	if in.NewEntry != nil {
		for i, t := range modeTabs {
			if t.mode == in.NewEntry.Mode {
				m.tab = i
			}
		}
	}

	m.refresh()
	return m
}

func (m *LeaderboardModel) Init() tea.Cmd {
	return nil
}

func (m *LeaderboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if m.search.Focused() {
			return m, m.searchUpdate(msg)
		}
		return m, m.keyUpdate(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	}
	return m, nil
}

func (m *LeaderboardModel) searchUpdate(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "enter", "up", "down":
		m.search.Blur()
		return nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.refresh()
	return cmd
}

func (m *LeaderboardModel) keyUpdate(msg tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Exit):
		// Esc clears an active search before it leaves the screen.
		if m.search.Value() != "" {
			m.search.SetValue("")
			m.refresh()
			return nil
		}
		return m.toMenu()
	case key.Matches(msg, m.keys.Back):
		return m.toMenu()
	case key.Matches(msg, m.keys.Replay):
		return tui.SwitchModeCmd(m.replay.Mode, m.replay)
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	case key.Matches(msg, m.keys.Search):
		return m.search.Focus()

	case key.Matches(msg, m.keys.Up):
		m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		m.moveCursor(1)
	case key.Matches(msg, m.keys.PageUp):
		m.moveCursor(-m.visibleRows())
	case key.Matches(msg, m.keys.PageDown):
		m.moveCursor(m.visibleRows())
	case key.Matches(msg, m.keys.Top):
		m.moveCursor(-len(m.rows))
	case key.Matches(msg, m.keys.Bottom):
		m.moveCursor(len(m.rows))

	case key.Matches(msg, m.keys.NextMode):
		m.tab = (m.tab + 1) % len(modeTabs)
		m.refresh()
	case key.Matches(msg, m.keys.PrevMode):
		m.tab = (m.tab + len(modeTabs) - 1) % len(modeTabs)
		m.refresh()

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
	return nil
}

func (m *LeaderboardModel) toMenu() tea.Cmd {
	in := tui.NewMenuInput()
	if m.replay != nil {
		in.Username, in.Mode, in.Level = m.replay.Username, m.replay.Mode, m.replay.Level
	}
	return tui.SwitchModeCmd(tui.ModeMenu, in)
}

func (m *LeaderboardModel) View() tea.View {
	header, footer := m.chrome()
	content := lipgloss.JoinVertical(lipgloss.Center, header, "", m.tableView(m.tableWidth()), "", footer)
	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

// chrome renders everything around the table.
func (m *LeaderboardModel) chrome() (header, footer string) {
	top := []string{theme.Title.Render("L E A D E R B O A R D"), "", m.tabsView()}
	if banner := m.bannerView(); banner != "" {
		top = append(top, "", banner)
	}
	header = lipgloss.JoinVertical(lipgloss.Center, top...)
	footer = lipgloss.JoinVertical(lipgloss.Center, m.statusView(), "", m.helpView())
	return header, footer
}

func (m *LeaderboardModel) tabsView() string {
	tabs := make([]string, len(modeTabs))
	for i, t := range modeTabs {
		if i == m.tab {
			tabs[i] = theme.Badge.Render(t.label)
		} else {
			tabs[i] = theme.Label.Padding(0, 1).Render(t.label)
		}
	}
	return strings.Join(tabs, " ")
}

// bannerView summarises the game that was just played, if any.
func (m *LeaderboardModel) bannerView() string {
	if m.newEntry == nil {
		return ""
	}
	rank, of := 1, 0
	for _, e := range m.entries {
		if e.Mode != m.newEntry.Mode {
			continue
		}
		of++
		if e.Score > m.newEntry.Score {
			rank++
		}
	}

	place := theme.Value.Render(fmt.Sprintf("#%d", rank))
	if rank <= 3 {
		place = lipgloss.NewStyle().Foreground(medalColor(rank)).Bold(true).Render(fmt.Sprintf("#%d", rank))
	}
	return theme.Label.Render("You scored ") +
		theme.Score.Render(strconv.Itoa(m.newEntry.Score)) +
		theme.Label.Render(" — ") + place +
		theme.Label.Render(fmt.Sprintf(" of %d in %s", of, modeLabel(m.newEntry.Mode)))
}

func (m *LeaderboardModel) statusView() string {
	if m.search.Focused() {
		return m.search.View()
	}

	dir := "↓"
	if !m.sort.desc {
		dir = "↑"
	}
	status := fmt.Sprintf("%d entries · sorted by %s %s", len(m.rows), sortColumnNames[m.sort.col], dir)
	if q := m.search.Value(); q != "" {
		status = fmt.Sprintf("%d matching %q · esc to clear", len(m.rows), q)
	}
	return theme.Hint.Render(status)
}

func (m *LeaderboardModel) helpView() string {
	if m.search.Focused() {
		return m.help.View(keyList{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "done")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "close")),
		})
	}
	return m.help.View(m.keys)
}

// column describes one table column for the current width.
type column struct {
	title string
	width int
	sort  sortColumn
	right bool
}

func (m *LeaderboardModel) columns(tableW int) []column {
	cols := []column{
		{"#", colRankW, -1, true},
		{"Name", 0, sortByName, false},
		{"Score", colScoreW, sortByScore, true},
		{"Level", colLevelW, sortByLevel, true},
		{"Mode", colModeW, sortByMode, false},
		{"Date", colDateW, sortByDate, false},
	}
	// Every row shares the same mode when a single mode is selected.
	if modeTabs[m.tab].mode != "" {
		cols = slices.DeleteFunc(cols, func(c column) bool { return c.sort == sortByMode })
	}

	fixed := func() int {
		w := colGap * (len(cols) - 1)
		for _, c := range cols {
			w += c.width
		}
		return w
	}
	// Drop the least useful columns on narrow terminals.
	for len(cols) > 3 && tableW-fixed() < minNameW {
		cols = cols[:len(cols)-1]
	}
	cols[1].width = max(tableW-fixed(), minNameW)
	return cols
}

func (m *LeaderboardModel) tableWidth() int {
	if m.width == 0 {
		return maxTableW
	}
	return min(m.width-4, maxTableW)
}

func (m *LeaderboardModel) tableView(tableW int) string {
	cols := m.columns(tableW)
	tableW = 0
	for i, c := range cols {
		tableW += c.width
		if i > 0 {
			tableW += colGap
		}
	}

	headerCells := make([]string, len(cols))
	for i, c := range cols {
		title := c.title
		style := theme.Label.Bold(true)
		if c.sort == m.sort.col {
			title += map[bool]string{true: " ↓", false: " ↑"}[m.sort.desc]
			style = theme.Key.Bold(true)
		}
		headerCells[i] = style.Render(pad(title, c.width, c.right))
	}
	lines := []string{
		strings.Join(headerCells, strings.Repeat(" ", colGap)),
		lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", tableW)),
	}

	if len(m.rows) == 0 {
		msg := "No scores yet — play a round to get on the board!"
		if m.search.Value() != "" {
			msg = fmt.Sprintf("Nobody matches %q.", m.search.Value())
		} else if modeTabs[m.tab].mode != "" {
			msg = "No " + modeTabs[m.tab].label + " games yet."
		}
		empty := lipgloss.Place(tableW, minVisible, lipgloss.Center, lipgloss.Center, theme.Hint.Render(msg))
		return lipgloss.JoinVertical(lipgloss.Left, append(lines, empty)...)
	}

	end := min(m.offset+m.visibleRows(), len(m.rows))
	for i := m.offset; i < end; i++ {
		lines = append(lines, m.rowView(m.rows[i], i == m.cursor, cols))
	}

	// Scroll position, when the list doesn't fit.
	if len(m.rows) > m.visibleRows() {
		lines = append(lines, theme.Hint.Width(tableW).Align(lipgloss.Right).
			Render(fmt.Sprintf("%d–%d of %d", m.offset+1, end, len(m.rows))))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *LeaderboardModel) rowView(e data.LeaderboardEntry, selected bool, cols []column) string {
	base := lipgloss.NewStyle()
	if selected {
		base = base.Background(theme.Highlight)
	}
	isNew := m.newEntry != nil && e.ID == m.newEntry.ID

	rank := m.ranks[e.ID]
	rankStyle := theme.Hint
	if rank >= 1 && rank <= 3 {
		rankStyle = lipgloss.NewStyle().Foreground(medalColor(rank)).Bold(true)
	}

	name := e.Name
	nameStyle := lipgloss.NewStyle()
	if isNew {
		name = "★ " + name
		nameStyle = theme.Title
	}

	date := e.CreatedAt
	if len(date) >= 10 {
		date = date[:10]
	}

	values := map[string]struct {
		text  string
		style lipgloss.Style
	}{
		"#":     {strconv.Itoa(rank), rankStyle},
		"Name":  {name, nameStyle},
		"Score": {strconv.Itoa(e.Score), theme.Score},
		"Level": {strconv.Itoa(e.Level), lipgloss.NewStyle()},
		"Mode":  {modeLabel(e.Mode), theme.Label},
		"Date":  {date, theme.Hint},
	}

	gap := base.Render(strings.Repeat(" ", colGap))
	cells := make([]string, len(cols))
	for i, c := range cols {
		v := values[c.title]
		cells[i] = v.style.Inherit(base).Render(pad(truncate(v.text, c.width), c.width, c.right))
	}
	return strings.Join(cells, gap)
}

func medalColor(rank int) color.Color {
	switch rank {
	case 1:
		return theme.Gold
	case 2:
		return theme.Silver
	default:
		return theme.Bronze
	}
}

func modeLabel(mode data.GameMode) string {
	switch mode {
	case data.GameModeNormal:
		return "Normal"
	case data.GameModeCrazy:
		return "Crazy"
	case data.GameModeAI:
		return "Versus AI"
	case "":
		return "—"
	default:
		return string(mode)
	}
}

// visibleRows is how many table rows fit between the header and footer.
func (m *LeaderboardModel) visibleRows() int {
	if m.height == 0 {
		return 10
	}
	header, footer := m.chrome()
	rows := m.height - lipgloss.Height(header) - lipgloss.Height(footer) - 6 // gaps + table header
	// Reserve a line for the scroll indicator when the list overflows.
	if len(m.rows) > rows {
		rows--
	}
	return max(rows, minVisible)
}

func (m *LeaderboardModel) moveCursor(delta int) {
	if len(m.rows) == 0 {
		return
	}
	m.cursor = min(max(m.cursor+delta, 0), len(m.rows)-1)
	m.clampScroll()
}

// clampScroll keeps the cursor inside the visible window.
func (m *LeaderboardModel) clampScroll() {
	n := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+n {
		m.offset = m.cursor - n + 1
	}
	m.offset = min(max(m.offset, 0), max(len(m.rows)-n, 0))
}

func (m *LeaderboardModel) toggleSort(col sortColumn) {
	if m.sort.col == col {
		m.sort.desc = !m.sort.desc
	} else {
		m.sort.col = col
		// Names and modes read best A→Z; numbers and dates best-first.
		m.sort.desc = col != sortByName && col != sortByMode
	}
	m.refresh()
}

// refresh rebuilds the visible rows and ranks, keeping the cursor on the same
// entry where possible (or on the new entry the first time).
func (m *LeaderboardModel) refresh() {
	selectedID := -1
	if m.cursor < len(m.rows) {
		selectedID = m.rows[m.cursor].ID
	} else if m.newEntry != nil {
		selectedID = m.newEntry.ID
	}

	mode := modeTabs[m.tab].mode
	inMode := make([]data.LeaderboardEntry, 0, len(m.entries))
	for _, e := range m.entries {
		if mode == "" || e.Mode == mode {
			inMode = append(inMode, e)
		}
	}

	// Competition ranking by score: ties share a rank.
	byScore := slices.Clone(inMode)
	slices.SortStableFunc(byScore, func(a, b data.LeaderboardEntry) int { return cmp.Compare(b.Score, a.Score) })
	m.ranks = make(map[int]int, len(byScore))
	for i, e := range byScore {
		if i > 0 && e.Score == byScore[i-1].Score {
			m.ranks[e.ID] = m.ranks[byScore[i-1].ID]
		} else {
			m.ranks[e.ID] = i + 1
		}
	}

	m.rows = m.filteredSorted(inMode)

	m.cursor = 0
	for i, e := range m.rows {
		if e.ID == selectedID {
			m.cursor = i
		}
	}
	m.clampScroll()
}

func (m *LeaderboardModel) filteredSorted(entries []data.LeaderboardEntry) []data.LeaderboardEntry {
	needle := strings.ToLower(strings.TrimSpace(m.search.Value()))

	filtered := make([]data.LeaderboardEntry, 0, len(entries))
	for _, e := range entries {
		if needle == "" || strings.Contains(strings.ToLower(e.Name), needle) {
			filtered = append(filtered, e)
		}
	}

	slices.SortFunc(filtered, func(a, b data.LeaderboardEntry) int {
		var c int

		switch m.sort.col {
		case sortByScore:
			c = cmp.Compare(a.Score, b.Score)
		case sortByLevel:
			c = cmp.Compare(a.Level, b.Level)
		case sortByName:
			c = cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		case sortByMode:
			c = cmp.Compare(strings.ToLower(string(a.Mode)), strings.ToLower(string(b.Mode)))
		case sortByDate:
			c = cmp.Compare(a.CreatedAt, b.CreatedAt)
		}

		if m.sort.desc {
			c = -c
		}

		// Ties fall back to the best score, then the newest entry.
		return cmp.Or(c, cmp.Compare(b.Score, a.Score), cmp.Compare(b.ID, a.ID))
	})

	return filtered
}

func uniqueNames(entries []data.LeaderboardEntry) []string {
	seen := make(map[string]struct{}, len(entries))
	names := make([]string, 0, len(entries))

	for _, e := range entries {
		if _, ok := seen[e.Name]; !ok {
			seen[e.Name] = struct{}{}
			names = append(names, e.Name)
		}
	}

	return names
}
