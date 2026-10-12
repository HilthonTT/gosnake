package views

import (
	"context"
	"fmt"
	"image/color"
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
	"github.com/HilthonTT/gosnake/internal/tui/theme"
	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/ai"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/crazy"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/single"
)

const (
	// TimerUpdateInterval is how often the elapsed-time display refreshes.
	TimerUpdateInterval = 100 * time.Millisecond

	scoreSubmitTimeout = 10 * time.Second

	sidebarWidth = 24
)

// Board pixel colours.
var (
	colBoardA   = lipgloss.Color("#142016")
	colBoardB   = lipgloss.Color("#18261A")
	colWall     = lipgloss.Color("#34453A")
	colHead     = lipgloss.Color("#D9F99D")
	colFood     = theme.Gold
	colBomb     = lipgloss.Color("#EF4444")
	colBombFuse = lipgloss.Color("#9F1239")
	colAIHead   = lipgloss.Color("#7DD3FC")
	colAIBody   = lipgloss.Color("#0284C7")

	// bodyRamp fades the player's body from just behind the head to the tail.
	bodyRamp = lipgloss.Blend1D(16, theme.Accent, lipgloss.Color("#3F6212"))
)

var sidebarStyle = theme.Panel.
	Width(sidebarWidth).
	Padding(1, 1)

var modalStyle = theme.Panel.
	Padding(1, 4).
	Align(lipgloss.Center)

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

	// BestScore is the local record for this mode, shown as the score to beat.
	BestScore int
}

var _ tea.Model = &SingleModel{}

type SingleModel struct {
	username string
	game     snake.GameController
	mode     tui.Mode

	// started is false until the player presses a direction, so the board
	// doesn't start moving before they've had a chance to look at it.
	started bool

	// tickStopwatch drives the snake's movement; its interval shrinks as the
	// level rises.
	tickStopwatch components.Stopwatch

	// gameStopwatch measures total elapsed play time for the info panel.
	gameStopwatch components.Stopwatch

	help help.Model
	keys *components.GameKeyMap

	submitter ScoreSubmitter
	bestScore int

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
		help:          theme.NewHelp(),
		game:          g,
		keys:          components.NewGameKeyMap(deps.Keys),
		tickStopwatch: components.NewStopwatchWithInterval(g.GetTickInterval()),
		gameStopwatch: components.NewStopwatchWithInterval(TimerUpdateInterval),
		submitter:     deps.Submitter,
		bestScore:     deps.BestScore,
		mode:          in.Mode,
	}, nil
}

// Init does not start the clocks; the first direction key does.
func (m *SingleModel) Init() tea.Cmd {
	return nil
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
		// Don't let the snake run blind while the board can't be seen.
		if _, fits := m.layout(); !fits && m.isRunning() {
			cmds = append(cmds, m.togglePause())
		}
		return m, tea.Batch(cmds...)
	}

	// Route to the correct state handler.
	switch {
	case m.game.IsGameOver():
		m, cmd = m.gameOverUpdate(msg)
	case !m.started:
		m, cmd = m.readyUpdate(msg)
	case m.game.IsPaused():
		m, cmd = m.pausedUpdate(msg)
	default:
		m, cmd = m.playingUpdate(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SingleModel) isRunning() bool {
	return m.started && !m.game.IsPaused() && !m.game.IsGameOver()
}

// layout picks the largest board scale that fits the terminal.
func (m *SingleModel) layout() (components.Scale, bool) {
	if m.width == 0 && m.height == 0 {
		return components.ScaleCompact, true // size not known yet
	}
	for _, scale := range []components.Scale{components.ScaleLarge, components.ScaleCompact} {
		w, h := m.requiredSize(scale)
		if m.width >= w && m.height >= h {
			return scale, true
		}
	}
	return components.ScaleCompact, false
}

func (m *SingleModel) requiredSize(scale components.Scale) (int, int) {
	bw, bh := components.BoardSize(snake.DefaultRows, snake.DefaultCols, scale)
	return sidebarWidth + 1 + bw, bh + 2 // gap column; blank line + help line
}

func (m *SingleModel) View() tea.View {
	scale, fits := m.layout()
	if !fits {
		w, h := m.requiredSize(components.ScaleCompact)
		return tea.NewView(theme.TooSmall(m.width, m.height, w, h))
	}

	board := m.boardView(scale)
	if modal := m.modalView(); modal != "" {
		board = components.Overlay(board, modal)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Top,
		m.sidebarView(lipgloss.Height(board)),
		" ",
		board,
	)
	content = lipgloss.JoinVertical(lipgloss.Center,
		content,
		"",
		m.helpView(),
	)

	return tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content))
}

// keyList is a help.KeyMap over a fixed set of bindings.
type keyList []key.Binding

func (k keyList) ShortHelp() []key.Binding  { return k }
func (k keyList) FullHelp() [][]key.Binding { return [][]key.Binding{k} }

// relabel returns a copy of b with a different help description.
func relabel(b key.Binding, desc string) key.Binding {
	b.SetHelp(b.Help().Key, desc)
	return b
}

// helpView shows the keys that do something in the current state, or every
// binding when the full help is toggled on.
func (m *SingleModel) helpView() string {
	if m.help.ShowAll {
		return m.help.View(m.keys)
	}

	var keys keyList
	switch {
	case m.game.IsGameOver():
		keys = keyList{relabel(m.keys.Submit, "leaderboard")}
	case !m.started:
		keys = keyList{relabel(m.keys.Move, "start"), m.keys.Quit}
	case m.game.IsPaused():
		keys = keyList{relabel(m.keys.Pause, "resume"), m.keys.Quit}
	default:
		keys = keyList{m.keys.Move, m.keys.Pause}
	}
	return m.help.View(append(keys, m.keys.Help))
}

// dependenciesUpdate keeps both stopwatches ticking regardless of game state.
func (m *SingleModel) dependenciesUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	var tickCmd, gameCmd tea.Cmd
	m.tickStopwatch, tickCmd = m.tickStopwatch.Update(msg)
	m.gameStopwatch, gameCmd = m.gameStopwatch.Update(msg)

	return m, tea.Batch(tickCmd, gameCmd)
}

func (m *SingleModel) gameOverUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok && (key.Matches(msg, m.keys.Submit) || key.Matches(msg, m.keys.Quit)) {
		newEntry := &data.LeaderboardEntry{
			Name:      m.username,
			Score:     m.game.Score(),
			Level:     m.game.Level(),
			CreatedAt: time.Now().Format(time.DateTime),
			Mode:      GameModeFromTUI(m.mode),
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

// direction maps a key press to a movement direction.
func (m *SingleModel) direction(msg tea.KeyPressMsg) (snake.Direction, bool) {
	switch {
	case key.Matches(msg, m.keys.Up):
		return snake.Up, true
	case key.Matches(msg, m.keys.Down):
		return snake.Down, true
	case key.Matches(msg, m.keys.Left):
		return snake.Left, true
	case key.Matches(msg, m.keys.Right):
		return snake.Right, true
	}
	return 0, false
}

func (m *SingleModel) readyUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	msgKey, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	if key.Matches(msgKey, m.keys.Quit) {
		return m, tea.Quit
	}
	if _, fits := m.layout(); !fits {
		return m, nil
	}
	dir, isDir := m.direction(msgKey)
	if !isDir && !key.Matches(msgKey, m.keys.Submit) {
		return m, nil
	}
	if isDir {
		m.game.ChangeDirection(dir)
	}
	m.started = true
	return m, tea.Batch(m.tickStopwatch.Init(), m.gameStopwatch.Init())
}

func (m *SingleModel) playingKeyUpdate(msg tea.KeyPressMsg) (*SingleModel, tea.Cmd) {
	if dir, ok := m.direction(msg); ok {
		m.game.ChangeDirection(dir)
		return m, nil
	}
	// Quit pauses first, so a stray key press can't throw away a run.
	if key.Matches(msg, m.keys.Pause) || key.Matches(msg, m.keys.Quit) {
		return m, m.togglePause()
	}
	return m, nil
}

func (m *SingleModel) pausedUpdate(msg tea.Msg) (*SingleModel, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(msg, m.keys.Pause), key.Matches(msg, m.keys.Submit):
			if _, fits := m.layout(); fits {
				return m, m.togglePause()
			}
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

func (m *SingleModel) boardView(scale components.Scale) string {
	matrix := m.game.Matrix()

	// Index each body segment so the body can fade from head to tail.
	body := m.game.Snake()
	segment := make(map[snake.Point]int, len(body))
	for i, p := range body {
		segment[p] = i
	}

	blinkOn := (time.Now().UnixMilli()/crazy.BombBlinkPeriodMs)%2 == 0
	dim := !m.isRunning()

	shade := func(c color.Color) color.Color {
		if dim && c != nil {
			return lipgloss.Darken(c, 0.5)
		}
		return c
	}
	colors := components.BoardColors{
		Wall: shade(colWall),
		Empty: func(row, col int) color.Color {
			// A subtle checkerboard makes distances easier to judge.
			if (row+col)%2 == 1 {
				return shade(colBoardB)
			}
			return shade(colBoardA)
		},
	}

	return components.RenderBoard(len(matrix), len(matrix[0]), scale, colors, func(row, col int) color.Color {
		var c color.Color
		switch matrix[row][col] {
		case snake.CellHead:
			c = colHead
		case snake.CellBody:
			i := segment[snake.Point{X: col, Y: row}]
			c = bodyRamp[(i*(len(bodyRamp)-1))/max(len(body)-1, 1)]
		case snake.CellFood:
			c = colFood
		case snake.CellBomb:
			c = colBomb
		case snake.CellBombWarning:
			if blinkOn || dim {
				c = colBombFuse
			}
		case snake.CellAIHead:
			c = colAIHead
		case snake.CellAIBody:
			c = colAIBody
		}
		return shade(c)
	})
}

// modalView is the dialog drawn over the board, if any.
func (m *SingleModel) modalView() string {

	switch {
	case m.game.IsGameOver():
		title, accent := "GAME OVER", theme.Danger
		if vs, ok := m.game.(snake.AIGameController); ok {
			switch vs.Outcome() {
			case snake.OutcomeWin:
				title, accent = "YOU WIN", theme.Accent
			case snake.OutcomeDraw:
				title, accent = "DRAW", theme.Warning
			}
		}

		const w = 20
		lines := []string{lipgloss.NewStyle().Foreground(accent).Bold(true).Render(title)}
		if result := m.resultLine(); result != "" {
			lines = append(lines, theme.Label.Render(result))
		}
		lines = append(lines, "",
			theme.SpaceBetween(w, theme.Label.Render("Score"), theme.Score.Render(fmt.Sprint(m.game.Score()))),
			theme.SpaceBetween(w, theme.Label.Render("Level"), theme.Value.Render(fmt.Sprint(m.game.Level()))),
			theme.SpaceBetween(w, theme.Label.Render("Length"), theme.Value.Render(fmt.Sprint(m.game.SnakeLength()))),
			theme.SpaceBetween(w, theme.Label.Render("Time"), theme.Value.Render(formatElapsed(m.gameStopwatch.Elapsed()))),
		)
		if m.isNewBest() {
			lines = append(lines, "", theme.Title.Render("★ new record! ★"))
		}
		lines = append(lines, "", keyRows(relabel(m.keys.Submit, "leaderboard")))
		return modalStyle.BorderForeground(accent).Render(lipgloss.JoinVertical(lipgloss.Center, lines...))

	case !m.started:
		greeting := "READY?"
		if m.username != "" {
			greeting = "READY, " + strings.ToUpper(truncate(m.username, 12)) + "?"
		}
		return modalStyle.BorderForeground(theme.Accent).Render(lipgloss.JoinVertical(lipgloss.Center,
			theme.Title.Render(greeting),
			"",
			keyRows(m.keys.Move, m.keys.Pause),
			"",
			theme.Hint.Render("press a direction to start"),
		))

	case m.game.IsPaused():
		return modalStyle.BorderForeground(theme.Warning).Render(lipgloss.JoinVertical(lipgloss.Center,
			lipgloss.NewStyle().Foreground(theme.Warning).Bold(true).Render("PAUSED"),
			"",
			keyRows(relabel(m.keys.Pause, "resume"), m.keys.Quit),
		))
	}
	return ""
}

// keyRows lists bindings as aligned "key  description" lines.
func keyRows(bindings ...key.Binding) string {
	w := 0
	for _, b := range bindings {
		w = max(w, lipgloss.Width(b.Help().Key))
	}
	lines := make([]string, len(bindings))
	for i, b := range bindings {
		lines[i] = theme.Key.Render(pad(b.Help().Key, w+2, false)) + theme.Hint.Render(b.Help().Desc)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func modeBadge(mode tui.Mode) string {
	if mode == tui.ModeAI {
		return "VS AI"
	}
	return strings.ToUpper(mode.String())
}

func (m *SingleModel) isNewBest() bool {
	return m.game.Score() > 0 && m.game.Score() > m.bestScore
}

// resultLine describes how a versus game ended; empty for solo modes.
func (m *SingleModel) resultLine() string {
	vs, ok := m.game.(snake.AIGameController)
	if !ok {
		return ""
	}
	switch vs.Outcome() {
	case snake.OutcomeWin:
		return "You outlasted the AI."
	case snake.OutcomeLoss:
		return "The AI got you."
	case snake.OutcomeDraw:
		return "Head-on collision."
	default:
		return ""
	}
}

func (m *SingleModel) sidebarView(height int) string {
	frameW, frameH := sidebarStyle.GetFrameSize()
	w := sidebarWidth - frameW

	row := func(label, value string) string {
		return theme.SpaceBetween(w, theme.Label.Render(label), value)
	}
	num := func(n int) string { return theme.Value.Render(fmt.Sprint(n)) }

	score := m.game.Score()
	best := max(m.bestScore, score)
	bestStyle := theme.Value
	if m.isNewBest() {
		bestStyle = theme.Title
	}

	level := m.game.Level()
	progress := 1.0
	if level < snake.MaxLevel {
		progress = float64(score%snake.PointsPerLevel) / snake.PointsPerLevel
	}

	lines := []string{
		theme.Badge.Render(modeBadge(m.mode)),
		theme.Label.Render(truncate(m.username, w)),
		"",
		row("Score", theme.Score.Render(fmt.Sprint(score))),
		row("Best", bestStyle.Render(fmt.Sprint(best))),
		row("Level", theme.Value.Render(fmt.Sprintf("%d", level))+theme.Hint.Render(fmt.Sprintf("/%d", snake.MaxLevel))),
		theme.ProgressBar(w, progress),
		row("Length", num(m.game.SnakeLength())),
		row("Time", theme.Value.Render(formatElapsed(m.gameStopwatch.Elapsed()))),
	}

	if vs, ok := m.game.(snake.AIGameController); ok {
		status := lipgloss.NewStyle().Foreground(theme.Accent).Render("● alive")
		if !vs.IsAIAlive() {
			status = lipgloss.NewStyle().Foreground(theme.Danger).Render("✕ crashed")
		}
		lines = append(lines,
			"",
			lipgloss.NewStyle().Foreground(colAIHead).Bold(true).Render("AI ")+
				lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", w-3)),
			row("Score", theme.Score.Render(fmt.Sprint(vs.AIScore()))),
			row("Length", num(vs.AISnakeLength())),
			row("Status", status),
		)
	}

	lines = append(lines, "")
	legend := m.legend()
	inner := height - frameH
	if gap := inner - len(lines) - len(legend); gap > 0 {
		lines = append(lines, make([]string, gap)...) // push the legend to the bottom
	}
	lines = append(lines, legend...)

	return sidebarStyle.Height(height).Render(strings.Join(lines, "\n"))
}

// legend explains the board colours used by the current mode.
func (m *SingleModel) legend() []string {
	swatch := func(c color.Color, label string) string {
		return lipgloss.NewStyle().Foreground(c).Render("■ ") + theme.Hint.Render(fmt.Sprintf("%-7s", label))
	}

	items := []string{swatch(theme.Accent, "you"), swatch(colFood, "food")}
	switch m.mode {
	case tui.ModeCrazy:
		items = append(items, swatch(colBomb, "bomb"), swatch(colBombFuse, "fuse"))
	case tui.ModeAI:
		items = append(items, swatch(colAIBody, "ai"))
	}

	var lines []string
	for i := 0; i < len(items); i += 2 {
		lines = append(lines, strings.Join(items[i:min(i+2, len(items))], " "))
	}
	return lines
}

// truncate shortens s to at most w columns, adding an ellipsis if cut.
func truncate(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	return string(r) + "…"
}

// pad aligns s within w columns.
func pad(s string, w int, right bool) string {
	fill := strings.Repeat(" ", max(w-lipgloss.Width(s), 0))
	if right {
		return fill + s
	}
	return s + fill
}

// formatElapsed renders "M:SS".
func formatElapsed(d time.Duration) string {
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
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

// GameModeFromTUI maps a TUI mode to the mode stored with leaderboard entries.
func GameModeFromTUI(m tui.Mode) data.GameMode {
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
