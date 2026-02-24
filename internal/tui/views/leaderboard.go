package views

import (
	"database/sql"
	"strconv"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var _ tea.Model = &LeaderboardModel{}

type LeaderboardModel struct {
	keys *leaderboardKeyMap
	help help.Model

	table table.Model
	repo  *data.LeaderboardRepository

	width  int
	height int
}

func NewLeaderboardModel(in *tui.LeaderboardInput, db *sql.DB) (*LeaderboardModel, error) {
	repo := data.NewLeaderboardRepository(db)

	var err error
	newEntryID := 0
	if in.NewEntry != nil {
		if in.NewEntry.Name == "" {
			in.NewEntry.Name = "Anonymous"
		}

		newEntryID, err = repo.Save(in.NewEntry.Name, in.NewEntry.Score, in.NewEntry.Level)
		if err != nil {
			return nil, err
		}
	}

	entries, err := repo.All()
	if err != nil {
		return nil, err
	}

	return &LeaderboardModel{
		keys:  defaultLeaderboardKeyMap(),
		help:  help.New(),
		repo:  repo,
		table: buildLeaderboardTable(entries, newEntryID),
	}, nil
}

func (m *LeaderboardModel) Init() tea.Cmd {
	return nil
}

func (m *LeaderboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Exit):
			return m, tui.SwitchModeCmd(tui.ModeMenu, tui.NewMenuInput())
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)

	return m, cmd
}

func (m *LeaderboardModel) View() string {
	output := m.table.View() + "\n" + m.help.View(m.keys)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, output)
}

func buildLeaderboardTable(entries []data.LeaderboardEntry, focusID int) table.Model {
	cols := []table.Column{
		{Title: "Name", Width: 10},
		{Title: "Score", Width: 10},
		{Title: "Level", Width: 10},
	}

	focusIndex := 0
	rows := make([]table.Row, len(entries))

	for i, e := range entries {
		if e.ID == focusID {
			focusIndex = i
		}

		rows[i] = table.Row{
			e.Name,
			strconv.Itoa(e.Score),
			strconv.Itoa(e.Level),
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
