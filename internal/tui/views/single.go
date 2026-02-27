package views

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/services/leaderboard"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/components"
	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/single"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/stopwatch"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	PausedMessage = `
    ____                            __  
   / __ \____ ___  __________  ____/ /  
  / /_/ / __ ^/ / / / ___/ _ \/ __  /   
 / ____/ /_/ / /_/ (__  )  __/ /_/ /    
/_/    \__,_/\__,_/____/\___/\__,_/     
Press PAUSE to continue or HOLD to exit.
`

	GameOverMessage = `
   ______                        ____                 
  / ____/___ _____ ___  ___     / __ \_   _____  _____
 / / __/ __ ^/ __ ^__ \/ _ \   / / / / | / / _ \/ ___/
/ /_/ / /_/ / / / / / /  __/  / /_/ /| |/ /  __/ /    
\____/\__,_/_/ /_/ /_/\___/   \____/ |___/\___/_/     

            Press EXIT or HOLD to continue.           
`

	TimerUpdateInterval = time.Millisecond * 13
)

var _ tea.Model = &SingleModel{}

type SingleModel struct {
	username string
	game     *single.Game

	// tickStopwatch drives the snake's movement; its interval shrinks as the
	// level rises
	tickStopwatch components.Stopwatch

	// gameStopwatch measures total elapsed play time for the info panel.
	gameStopwatch components.Stopwatch

	help   help.Model
	keys   *components.GameKeyMap
	styles *components.GameStyles

	leaderboardService *leaderboard.LeaderboardService

	width  int
	height int
}

func NewSingleModel(in *tui.SingleInput, db *sql.DB) (*SingleModel, error) {
	repo := data.NewLeaderboardRepository(db)

	g, err := single.NewGame(repo)
	if err != nil {
		return nil, fmt.Errorf("creating snake game")
	}

	m := &SingleModel{
		username:           in.Username,
		help:               help.New(),
		game:               g,
		keys:               components.NewGameKeyMap(),
		tickStopwatch:      components.NewStopwatchWithInterval(g.GetDefaultTickInterval()),
		gameStopwatch:      components.NewStopwatchWithInterval(TimerUpdateInterval),
		styles:             components.CreateGameStyles(),
		leaderboardService: leaderboard.NewLeaderboardService(),
	}

	return m, nil
}

func (m *SingleModel) Init() tea.Cmd {
	return tea.Batch(
		m.tickStopwatch.Init(),
		m.gameStopwatch.Init(),
	)
}

func (m *SingleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m, cmd = m.dependenciesUpdate(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.ForceQuit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
			return m, tea.Batch(cmds...)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, tea.Batch(cmds...)
	}

	// Route to the correct state handler.
	if m.game.IsGameOver() {
		m, cmd = m.gameOverUpdate(msg)
	} else if m.game.IsPaused() {
		m, cmd = m.pausedUpdate(msg)
	} else {
		m, cmd = m.playingUpdate(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SingleModel) View() string {
	var board string
	switch {
	case m.game.IsGameOver():
		board = m.overlayView(m.styles.Overlay.GameOver, GameOverMessage)
	case m.game.IsPaused():
		board = m.overlayView(m.styles.Overlay.Paused, PausedMessage)
	default:
		board = m.matrixView()
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		m.infoView(),
		board,
	)
	content = lipgloss.JoinVertical(lipgloss.Left,
		content,
		m.styles.Help.Render(m.help.View(m.keys)),
	)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// dependenciesUpdate keeps both stopwatches ticking regardless of game state.
func (m *SingleModel) dependenciesUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	var cmds []tea.Cmd

	tickModel, cmd := m.tickStopwatch.Update(msg)
	m.tickStopwatch = tickModel.(components.Stopwatch)
	cmds = append(cmds, cmd)

	gameModel, cmd := m.gameStopwatch.Update(msg)
	m.gameStopwatch = gameModel.(components.Stopwatch)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SingleModel) gameOverUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(msg, m.keys.Quit) {

			newEntry := &data.LeaderboardEntry{
				Name:      m.username,
				Score:     m.game.Score(),
				Level:     m.game.Level(),
				CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
			}

			return m, tui.SwitchModeCmd(
				tui.ModeLeaderboard,
				tui.NewLeaderboardInput(tui.WithNewEntry(newEntry)),
			)
		}
	}
	return m, nil
}

func (m *SingleModel) togglePause() tea.Cmd {
	m.game.TogglePause()
	return tea.Batch(
		m.gameStopwatch.Toggle(),
		m.tickStopwatch.Toggle(),
	)
}

func (m *SingleModel) playingKeyUpdate(msg tea.KeyMsg) (*SingleModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.game.ChangeDirection(single.Up)
	case key.Matches(msg, m.keys.Down):
		m.game.ChangeDirection(single.Down)
	case key.Matches(msg, m.keys.Left):
		m.game.ChangeDirection(single.Left)
	case key.Matches(msg, m.keys.Right):
		m.game.ChangeDirection(single.Right)
	case key.Matches(msg, m.keys.Pause):
		return m, m.togglePause()
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

func (m *SingleModel) pausedUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Pause):
			return m, m.togglePause()
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *SingleModel) playingUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.playingKeyUpdate(msg)

	case stopwatch.TickMsg:
		// Only react to our tick stopwatch, not the game-time one.
		if msg.ID != m.tickStopwatch.ID() {
			break
		}
		m.game.Tick()

		// Adjust tick speed to match the (possibly new) level.
		m.tickStopwatch.SetInterval(m.game.GetTickInterval())

		if m.game.IsGameOver() {
			m.submitScore()

			return m, tea.Batch(
				m.tickStopwatch.Stop(),
				m.gameStopwatch.Stop(),
			)
		}
	}

	return m, nil
}

func (m *SingleModel) matrixView() string {
	matrix := m.game.Matrix()

	var sb strings.Builder
	for row := range matrix {
		for col := range matrix[row] {
			sb.WriteString(m.renderCell(matrix[row][col]))
		}
		if row < len(matrix)-1 {
			sb.WriteByte('\n')
		}
	}

	return m.styles.Board.Render(sb.String())
}

func (m *SingleModel) overlayView(style lipgloss.Style, msg string) string {
	const (
		innerW = snake.DefaultCols * 2 // two chars per cell
		innerH = snake.DefaultRows
	)
	inner := lipgloss.Place(innerW, innerH,
		lipgloss.Center, lipgloss.Center,
		style.Render(msg),
	)
	return m.styles.Board.Render(inner)
}

func (m *SingleModel) renderCell(cell byte) string {
	chars := m.styles.CellChars
	switch cell {
	case 'H':
		return m.styles.HeadCell.Render(chars.Head)
	case 'S':
		return m.styles.BodyCell.Render(chars.Body)
	case 'F':
		return m.styles.FoodCell.Render(chars.Food)
	default:
		return m.styles.EmptyCell.Render(chars.Empty)
	}
}

func (m *SingleModel) infoView() string {
	s := m.styles.Info
	divider := s.Divider.Render(strings.Repeat("─", 12))

	var headerText string
	switch {
	case m.game.IsGameOver():
		headerText = "GAME OVER"
	case m.game.IsPaused():
		headerText = "PAUSED"
	default:
		headerText = "SNAKE ON"
	}
	header := s.Title.Render(headerText)

	elapsed := m.gameStopwatch.Elapsed().Seconds()
	minutes := int(elapsed) / 60
	seconds := int(elapsed) % 60
	centis := int(elapsed*100) % 100

	var timeStr string
	if minutes > 0 {
		timeStr = fmt.Sprintf("%02d:%02d", minutes, seconds)
	} else {
		timeStr = fmt.Sprintf("%02d.%02d", seconds, centis)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		s.SectionLbl.Render("Score"),
		s.ValueBig.Render(fmt.Sprintf("%d", m.game.Score())),
		"\n",
		divider,
		"\n",
		s.SectionLbl.Render("Level"),
		s.ValueBig.Render(fmt.Sprintf("%d", m.game.Level())),
		"\n",
		divider,
		"\n",
		s.SectionLbl.Render("Time"),
		s.ValueBig.Render(timeStr),
		"\n",
		divider,
		"\n",
		s.SectionLbl.Render("Length"),
		s.ValueBig.Render(fmt.Sprintf("%d", m.game.SnakeLength())),
	)

	return s.Panel.Render(body)
}

func (m *SingleModel) submitScore() {
	req := leaderboard.SubmitScoreRequest{
		PlayerName:  m.username,
		Score:       m.game.Score(),
		Level:       m.game.Level(),
		SnakeLength: m.game.SnakeLength(),
	}

	go func() {
		_, err := m.leaderboardService.SubmitScore(context.Background(), req)
		if err != nil {
			log.Printf("%s", err.Error())
		}
	}()
}
