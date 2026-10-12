package views

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/stopwatch"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/HilthonTT/gosnake/internal/config"
	"github.com/HilthonTT/gosnake/internal/data"
	"github.com/HilthonTT/gosnake/internal/services/leaderboard"
	"github.com/HilthonTT/gosnake/internal/telemetry"
	"github.com/HilthonTT/gosnake/internal/tui"
	"github.com/HilthonTT/gosnake/internal/tui/components"
	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/ai"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/crazy"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/single"
)

const (
	pausedBanner = `
    ____                            __
   / __ \____ ___  __________  ____/ /
  / /_/ / __ ^/ / / / ___/ _ \/ __  /
 / ____/ /_/ / /_/ (__  )  __/ /_/ /
/_/    \__,_/\__,_/____/\___/\__,_/`

	gameOverBanner = `
   ______                        ____
  / ____/___ _____ ___  ___     / __ \_   _____  _____
 / / __/ __ ^/ __ ^__ \/ _ \   / / / / | / / _ \/ ___/
/ /_/ / /_/ / / / / / /  __/  / /_/ /| |/ /  __/ /
\____/\__,_/_/ /_/ /_/\___/   \____/ |___/\___/_/`

	TimerUpdateInterval = time.Millisecond * 13

	scoreSubmitTimeout = 10 * time.Second
)

// ScoreSubmitter sends finished games to the online leaderboard.
type ScoreSubmitter interface {
	SubmitScore(ctx context.Context, req leaderboard.SubmitScoreRequest) (leaderboard.LeaderboardEntry, error)
}

// SingleDeps are the collaborators a SingleModel needs.
type SingleDeps struct {
	// Keys are the configured key bindings; nil uses the defaults.
	Keys *config.Keys

	// Submitter publishes the final score online; nil disables it.
	Submitter ScoreSubmitter
}

var _ tea.Model = &SingleModel{}

type SingleModel struct {
	username string
	game     snake.GameController
	mode     tui.Mode

	// tickStopwatch drives the snake's movement; its interval shrinks as the
	// level rises.
	tickStopwatch components.Stopwatch

	// gameStopwatch measures total elapsed play time for the info panel.
	gameStopwatch components.Stopwatch

	help   help.Model
	keys   *components.GameKeyMap
	styles *components.GameStyles

	submitter ScoreSubmitter

	width  int
	height int
}

// newGame creates the engine for the given mode.
func newGame(mode tui.Mode, level int) (snake.GameController, error) {
	switch mode {
	case tui.ModeNormal:
		return single.NewGame(level)
	case tui.ModeCrazy:
		return crazy.NewGame(level)
	case tui.ModeAI:
		return ai.NewGame(level)
	default:
		return nil, fmt.Errorf("unsupported game mode: %v", mode)
	}
}

func NewSingleModel(in *tui.SingleInput, deps SingleDeps) (*SingleModel, error) {
	level := max(in.Level, 1)

	g, err := newGame(in.Mode, level)
	if err != nil {
		return nil, fmt.Errorf("creating %s game: %w", in.Mode, err)
	}

	return &SingleModel{
		username:      in.Username,
		help:          help.New(),
		game:          g,
		keys:          components.NewGameKeyMap(deps.Keys),
		tickStopwatch: components.NewStopwatchWithInterval(g.GetTickInterval()),
		gameStopwatch: components.NewStopwatchWithInterval(TimerUpdateInterval),
		styles:        components.CreateGameStyles(),
		submitter:     deps.Submitter,
		mode:          in.Mode,
	}, nil
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
	case tea.KeyPressMsg:
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
	switch {
	case m.game.IsGameOver():
		m, cmd = m.gameOverUpdate(msg)
	case m.game.IsPaused():
		m, cmd = m.pausedUpdate(msg)
	default:
		m, cmd = m.playingUpdate(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SingleModel) View() tea.View {
	var board string
	switch {
	case m.game.IsGameOver():
		hint := fmt.Sprintf("Press %s to see the leaderboard.", strings.ToUpper(m.keys.Quit.Help().Key))
		board = m.overlayView(m.styles.Overlay.GameOver, gameOverBanner, m.resultLine(), hint)
	case m.game.IsPaused():
		hint := fmt.Sprintf("Press %s to resume or %s to quit.",
			strings.ToUpper(m.keys.Pause.Help().Key), strings.ToUpper(m.keys.Quit.Help().Key))
		board = m.overlayView(m.styles.Overlay.Paused, pausedBanner, hint)
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

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

// dependenciesUpdate keeps both stopwatches ticking regardless of game state.
func (m *SingleModel) dependenciesUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	var tickCmd, gameCmd tea.Cmd
	m.tickStopwatch, tickCmd = m.tickStopwatch.Update(msg)
	m.gameStopwatch, gameCmd = m.gameStopwatch.Update(msg)

	return m, tea.Batch(tickCmd, gameCmd)
}

func (m *SingleModel) gameOverUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok && key.Matches(msg, m.keys.Quit) {
		newEntry := &data.LeaderboardEntry{
			Name:      m.username,
			Score:     m.game.Score(),
			Level:     m.game.Level(),
			CreatedAt: time.Now().Format(time.DateTime),
			Mode:      gameModeFromTUI(m.mode),
		}

		return m, tui.SwitchModeCmd(
			tui.ModeLeaderboard,
			tui.NewLeaderboardInput(tui.WithNewEntry(newEntry)),
		)
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

func (m *SingleModel) playingKeyUpdate(msg tea.KeyPressMsg) (*SingleModel, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.game.ChangeDirection(snake.Up)
	case key.Matches(msg, m.keys.Down):
		m.game.ChangeDirection(snake.Down)
	case key.Matches(msg, m.keys.Left):
		m.game.ChangeDirection(snake.Left)
	case key.Matches(msg, m.keys.Right):
		m.game.ChangeDirection(snake.Right)
	case key.Matches(msg, m.keys.Pause):
		return m, m.togglePause()
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

func (m *SingleModel) pausedUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
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
	case tea.KeyPressMsg:
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

func (m *SingleModel) overlayView(style lipgloss.Style, banner string, lines ...string) string {
	const (
		innerW = snake.DefaultCols * 2 // two chars per cell
		innerH = snake.DefaultRows
	)

	text := lipgloss.JoinVertical(lipgloss.Center, append([]string{banner, ""}, lines...)...)
	inner := lipgloss.Place(innerW, innerH,
		lipgloss.Center, lipgloss.Center,
		style.Render(text),
	)
	return m.styles.Board.Render(inner)
}

// resultLine describes how a versus game ended; empty for solo modes.
func (m *SingleModel) resultLine() string {
	vs, ok := m.game.(snake.AIGameController)
	if !ok {
		return ""
	}
	switch vs.Outcome() {
	case snake.OutcomeWin:
		return "You outlasted the AI — you win!"
	case snake.OutcomeLoss:
		return "The AI got you — you lose."
	case snake.OutcomeDraw:
		return "Head-on collision — it's a draw."
	default:
		return ""
	}
}

func (m *SingleModel) renderCell(cell byte) string {
	chars := m.styles.CellChars
	switch cell {
	case snake.CellHead:
		return m.styles.HeadCell.Render(chars.Head)
	case snake.CellBody:
		return m.styles.BodyCell.Render(chars.Body)
	case snake.CellFood:
		return m.styles.FoodCell.Render(chars.Food)
	case snake.CellBomb:
		// Active bomb — always shown with a distinctive style.
		return m.styles.BombCell.Render(chars.Bomb)
	case snake.CellBombWarning:
		// Warning bomb — blinks by alternating between the warning glyph and
		// an empty cell based on wall-clock milliseconds.
		if (time.Now().UnixMilli()/crazy.BombBlinkPeriodMs)%2 == 0 {
			return m.styles.BombWarningCell.Render(chars.BombWarning)
		}
		return m.styles.EmptyCell.Render(chars.Empty)
	case snake.CellAIHead:
		return m.styles.AIHeadCell.Render(chars.AIHead)
	case snake.CellAIBody:
		return m.styles.AIBodyCell.Render(chars.AIBody)
	default:
		return m.styles.EmptyCell.Render(chars.Empty)
	}
}

func (m *SingleModel) infoView() string {
	s := m.styles.Info
	divider := s.Divider.Render(strings.Repeat("─", 12))

	vs, isVersus := m.game.(snake.AIGameController)

	var headerText string
	switch {
	case m.game.IsGameOver() && isVersus && vs.Outcome() == snake.OutcomeWin:
		headerText = "YOU WIN"
	case m.game.IsGameOver():
		headerText = "GAME OVER"
	case m.game.IsPaused():
		headerText = "PAUSED"
	case m.mode == tui.ModeCrazy:
		headerText = "CRAZY MODE"
	case isVersus:
		headerText = "VS AI"
	default:
		headerText = "SNAKE ON"
	}

	section := func(label, value string) []string {
		return []string{s.SectionLbl.Render(label), s.ValueBig.Render(value)}
	}
	join := func(sections ...[]string) string {
		var parts []string
		for i, sec := range sections {
			if i > 0 {
				parts = append(parts, "\n", divider, "\n")
			}
			parts = append(parts, sec...)
		}
		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		s.Title.Render(headerText),
		"\n",
		join(
			section("Score", fmt.Sprint(m.game.Score())),
			section("Level", fmt.Sprint(m.game.Level())),
			section("Time", formatElapsed(m.gameStopwatch.Elapsed())),
			section("Length", fmt.Sprint(m.game.SnakeLength())),
		),
	)

	if isVersus {
		aiStatus := "ALIVE"
		if !vs.IsAIAlive() {
			aiStatus = "DEAD"
		}

		body = lipgloss.JoinVertical(lipgloss.Left,
			body,
			"\n", divider, "\n",
			s.AILabel.Render("── AI ──"),
			"\n",
			join(
				section("Score", fmt.Sprint(vs.AIScore())),
				section("Length", fmt.Sprint(vs.AISnakeLength())),
				section("Status", aiStatus),
			),
		)
	}

	return s.Panel.Render(body)
}

// formatElapsed renders "SS.cc" under a minute and "MM:SS" afterwards.
func formatElapsed(d time.Duration) string {
	if d >= time.Minute {
		return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%02d.%02d", int(d.Seconds()), int(d.Milliseconds()/10)%100)
}

// submitScore publishes the result to the online leaderboard, if configured.
// It runs in the background so a slow or unreachable server never stalls the UI.
func (m *SingleModel) submitScore() {
	if m.submitter == nil {
		return
	}

	req := leaderboard.SubmitScoreRequest{
		PlayerName:  m.username,
		Score:       m.game.Score(),
		Level:       m.game.Level(),
		SnakeLength: m.game.SnakeLength(),
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), scoreSubmitTimeout)
		defer cancel()

		if _, err := m.submitter.SubmitScore(ctx, req); err != nil {
			slog.Warn("submitting score to online leaderboard", "err", err)
		}
	}()
}

func gameModeFromTUI(m tui.Mode) data.GameMode {
	switch m {
	case tui.ModeCrazy:
		return data.GameModeCrazy
	case tui.ModeAI:
		return data.GameModeAI
	default:
		return data.GameModeNormal
	}
}

func (m *SingleModel) GameSnapshot() map[string]any {
	if s, ok := m.game.(telemetry.Snapshottable); ok {
		return s.Snapshot()
	}
	return nil
}
