package server

import (
	"fmt"
	"strings"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/multi"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	colP0Head  = lipgloss.Color("82")  // bright lime — player 0
	colP0Body  = lipgloss.Color("40")  // medium green
	colP1Head  = lipgloss.Color("39")  // bright cyan — player 1
	colP1Body  = lipgloss.Color("37")  // teal
	colP2Head  = lipgloss.Color("208") // bright orange — player 2
	colP2Body  = lipgloss.Color("202") // deep orange
	colFood    = lipgloss.Color("196") // red
	colEmpty   = lipgloss.Color("236") // very dark grey
	colBorder  = lipgloss.Color("241") // mid grey
	colMuted   = lipgloss.Color("243")
	colScore   = lipgloss.Color("220") // warm yellow
	colTitle   = lipgloss.Color("82")  // matches P0 head
	colPause   = lipgloss.Color("214") // amber
	colDead    = lipgloss.Color("196") // red
	colWin     = lipgloss.Color("226") // bright yellow
	colWaiting = lipgloss.Color("244")
	colNote    = lipgloss.Color("248")
)

type multiStyles struct {
	board lipgloss.Style
	info  lipgloss.Style

	heads  [multi.MaxPlayers]lipgloss.Style
	bodies [multi.MaxPlayers]lipgloss.Style
	food   lipgloss.Style
	empty  lipgloss.Style

	title      lipgloss.Style
	sectionLbl lipgloss.Style
	valueBig   lipgloss.Style
	divider    lipgloss.Style

	overlayDead    lipgloss.Style
	overlayOver    lipgloss.Style
	overlayWaiting lipgloss.Style
	note           lipgloss.Style
}

func newMultiStyles() multiStyles {
	panelW := 22

	s := multiStyles{
		board: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder),

		info: lipgloss.NewStyle().
			Width(panelW).
			Padding(1, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder),

		food:  lipgloss.NewStyle().Foreground(colFood).Bold(true),
		empty: lipgloss.NewStyle().Foreground(colEmpty),

		title:      lipgloss.NewStyle().Width(panelW - 2).Align(lipgloss.Center).Bold(true).Foreground(colTitle),
		sectionLbl: lipgloss.NewStyle().Foreground(colMuted).Width(panelW - 2),
		valueBig:   lipgloss.NewStyle().Width(panelW - 2).Align(lipgloss.Right).Bold(true).Foreground(colScore),
		divider:    lipgloss.NewStyle().Foreground(colBorder).Width(panelW - 2),

		overlayDead:    lipgloss.NewStyle().Foreground(colDead).Bold(true).Padding(0, 2),
		overlayOver:    lipgloss.NewStyle().Foreground(colWin).Bold(true).Padding(0, 2),
		overlayWaiting: lipgloss.NewStyle().Foreground(colWaiting).Italic(true).Padding(0, 2),
		note:           lipgloss.NewStyle().Foreground(colNote).Italic(true),
	}

	// Per-player head / body styles.
	headColors := [multi.MaxPlayers]lipgloss.Color{colP0Head, colP1Head, colP2Head}
	bodyColors := [multi.MaxPlayers]lipgloss.Color{colP0Body, colP1Body, colP2Body}
	for i := range s.heads {
		s.heads[i] = lipgloss.NewStyle().Foreground(headColors[i]).Bold(true)
		s.bodies[i] = lipgloss.NewStyle().Foreground(bodyColors[i])
	}

	return s
}

type multiKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Restart   key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
}

func newMultiKeyMap() multiKeyMap {
	return multiKeyMap{
		Up:        key.NewBinding(key.WithKeys("up", "w", "k"), key.WithHelp("↑/w", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "s", "j"), key.WithHelp("↓/s", "down")),
		Left:      key.NewBinding(key.WithKeys("left", "a", "h"), key.WithHelp("←/a", "left")),
		Right:     key.NewBinding(key.WithKeys("right", "d", "l"), key.WithHelp("→/d", "right")),
		Restart:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart")),
		Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
		ForceQuit: key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "force quit")),
	}
}

var _ tea.Model = &SharedMultiGame{}

// SharedMultiGame is the Bubble Tea model that runs inside each player's SSH
// session.  It has no game logic of its own — it only renders incoming
// GameStateMsg snapshots and forwards key presses to the room via sync.
type SharedMultiGame struct {
	player *Player
	sync   chan tea.Msg
	keys   multiKeyMap
	styles multiStyles

	// Last received snapshot from the room.
	lastState *GameStateMsg

	// Per-session state.
	myDead bool   // true once we've received our own death notification
	note   string // last NoteMsg text

	width  int
	height int
}

func newSharedMultiGame(p *Player, sync chan tea.Msg) *SharedMultiGame {
	return &SharedMultiGame{
		player: p,
		sync:   sync,
		keys:   newMultiKeyMap(),
		styles: newMultiStyles(),
	}
}

func (m *SharedMultiGame) Init() tea.Cmd {
	return nil
}

func (m *SharedMultiGame) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case NoteMsg:
		m.note = string(msg)
		return m, nil

	case GameStateMsg:
		// Detect if this player died this tick.
		if m.player.playerIndex >= 0 && !m.myDead {
			for _, idx := range msg.Died {
				if idx == m.player.playerIndex {
					m.myDead = true
					break
				}
			}
		}
		m.lastState = &msg
		return m, nil

	case RestartMsg:
		// Room restarted — clear local state and wait for first tick.
		m.lastState = nil
		m.myDead = false
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.ForceQuit) || key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey routes keyboard input depending on current game state.
func (m *SharedMultiGame) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Game over — only restart is meaningful.
	if m.lastState != nil && m.lastState.Over {
		if key.Matches(msg, m.keys.Restart) {
			m.sendToRoom(RestartMsg{})
		}
		return m, nil
	}

	// Observers and dead players cannot steer.
	if m.player.playerIndex < 0 || m.myDead {
		return m, nil
	}

	// Active player — send direction changes to the room.
	idx := m.player.playerIndex
	switch {
	case key.Matches(msg, m.keys.Up):
		m.sendToRoom(DirectionMsg{PlayerIndex: idx, Direction: int(multi.Up)})
	case key.Matches(msg, m.keys.Down):
		m.sendToRoom(DirectionMsg{PlayerIndex: idx, Direction: int(multi.Down)})
	case key.Matches(msg, m.keys.Left):
		m.sendToRoom(DirectionMsg{PlayerIndex: idx, Direction: int(multi.Left)})
	case key.Matches(msg, m.keys.Right):
		m.sendToRoom(DirectionMsg{PlayerIndex: idx, Direction: int(multi.Right)})
	}
	return m, nil
}

// sendToRoom sends a message to the room's sync channel without blocking.
func (m *SharedMultiGame) sendToRoom(msg tea.Msg) {
	go func() {
		m.sync <- msg
	}()
}

func (m *SharedMultiGame) View() string {
	// No state yet — waiting for enough players to join.
	if m.lastState == nil {
		return m.waitingView()
	}

	board := m.boardView()
	info := m.infoView()

	content := lipgloss.JoinHorizontal(lipgloss.Top, info, board)
	if m.note != "" {
		content = lipgloss.JoinVertical(lipgloss.Left,
			content,
			m.styles.note.Render(" "+m.note),
		)
	}

	helpText := m.styles.sectionLbl.Render("↑↓←→/wasd move  q quit")
	if m.lastState.Over {
		helpText = m.styles.sectionLbl.Render("r restart  q quit")
	}
	content = lipgloss.JoinVertical(lipgloss.Left, content, helpText)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// boardView renders the matrix with overlays for dead / game-over states.
func (m *SharedMultiGame) boardView() string {
	state := m.lastState

	const (
		innerW = snake.DefaultCols * 2 // two chars per cell
		innerH = snake.DefaultRows
	)

	var inner string
	switch {
	case state.Over:
		inner = lipgloss.Place(innerW, innerH,
			lipgloss.Center, lipgloss.Center,
			m.styles.overlayOver.Render(m.gameOverMsg(state)),
		)
	case m.myDead:
		// Overlay on top of the live board so the spectator can still see action.
		board := m.matrixStr(state.Matrix)
		deadMsg := m.styles.overlayDead.Render(deadOverlay)
		// Place the overlay centred over the board string.
		inner = lipgloss.Place(innerW, innerH,
			lipgloss.Center, lipgloss.Center,
			lipgloss.JoinVertical(lipgloss.Center, board, deadMsg),
		)
	default:
		inner = m.matrixStr(state.Matrix)
	}

	return m.styles.board.Render(inner)
}

// waitingView is shown before the game starts (fewer than 2 players).
func (m *SharedMultiGame) waitingView() string {
	role := m.player.roleString()
	msg := fmt.Sprintf(
		"  MULTI SNAKE\n\n  Connected as %s\n\n  Waiting for players...\n  (need at least 2 to start)",
		role,
	)
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		m.styles.overlayWaiting.Render(msg),
	)
}

const deadOverlay = `
 ██╗   ██╗ ██████╗ ██╗   ██╗    ██████╗ ██╗███████╗██████╗ 
 ╚██╗ ██╔╝██╔═══██╗██║   ██║    ██╔══██╗██║██╔════╝██╔══██╗
  ╚████╔╝ ██║   ██║██║   ██║    ██║  ██║██║█████╗  ██║  ██║
   ╚██╔╝  ██║   ██║██║   ██║    ██║  ██║██║██╔══╝  ██║  ██║
    ██║   ╚██████╔╝╚██████╔╝    ██████╔╝██║███████╗██████╔╝
    ╚═╝    ╚═════╝  ╚═════╝     ╚═════╝ ╚═╝╚══════╝╚═════╝ 
              You are now spectating. Watch and learn.`

func (m *SharedMultiGame) gameOverMsg(state *GameStateMsg) string {
	var title string
	if state.Winner == -1 {
		title = "DRAW — everyone died!"
	} else if state.Winner == m.player.playerIndex {
		title = "🏆  YOU WIN!"
	} else {
		// Find winner name from snapshots.
		winName := fmt.Sprintf("Player %d", state.Winner+1)
		for _, s := range state.Players {
			if s.Index == state.Winner {
				winName = s.Name
				break
			}
		}
		title = fmt.Sprintf("%s wins!", winName)
	}

	return fmt.Sprintf(`
  ██████╗  █████╗ ███╗   ███╗███████╗
 ██╔════╝ ██╔══██╗████╗ ████║██╔════╝
 ██║  ███╗███████║██╔████╔██║█████╗  
 ██║   ██║██╔══██║██║╚██╔╝██║██╔══╝  
 ╚██████╔╝██║  ██║██║ ╚═╝ ██║███████╗
  ╚═════╝ ╚═╝  ╚═╝╚═╝     ╚═╝╚══════╝
           OVER

  %s

  Press R to restart  |  q to quit`, title)
}

// matrixStr converts a raw matrix into a styled two-char-per-cell string.
func (m *SharedMultiGame) matrixStr(matrix snake.Matrix) string {
	var sb strings.Builder
	for row := range matrix {
		for col := range matrix[row] {
			sb.WriteString(m.renderCell(matrix[row][col]))
		}
		if row < len(matrix)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// renderCell maps a matrix byte to a styled two-column string.
func (m *SharedMultiGame) renderCell(cell byte) string {
	s := m.styles
	switch cell {
	// Player 0
	case multi.HeadCells[0]:
		return s.heads[0].Render("██")
	case multi.BodyCells[0]:
		return s.bodies[0].Render("▓▓")
	// Player 1
	case multi.HeadCells[1]:
		return s.heads[1].Render("██")
	case multi.BodyCells[1]:
		return s.bodies[1].Render("▓▓")
	// Player 2
	case multi.HeadCells[2]:
		return s.heads[2].Render("██")
	case multi.BodyCells[2]:
		return s.bodies[2].Render("▓▓")
	// Food
	case 'F':
		return s.food.Render("◆ ")
	default:
		return s.empty.Render("· ")
	}
}

// infoView renders the side panel.
func (m *SharedMultiGame) infoView() string {
	s := m.styles
	state := m.lastState
	divider := s.divider.Render(strings.Repeat("─", 18))

	header := s.title.Render("MULTI SNAKE")

	// Player rows.
	var playerRows []string
	playerColors := [multi.MaxPlayers]lipgloss.Color{colP0Head, colP1Head, colP2Head}
	for _, snap := range state.Players {
		marker := "●"
		style := lipgloss.NewStyle().Foreground(playerColors[snap.Index]).Bold(true)
		if !snap.Alive {
			marker = "○"
			style = lipgloss.NewStyle().Foreground(colMuted)
		}

		tag := ""
		if snap.Index == m.player.playerIndex {
			tag = " ◀ you"
		}

		line := fmt.Sprintf("%s %s%s",
			style.Render(marker),
			snap.Name,
			lipgloss.NewStyle().Foreground(colMuted).Render(tag),
		)
		score := fmt.Sprintf("Score: %d  Len: %d", snap.Score, snap.Length)
		playerRows = append(playerRows,
			s.sectionLbl.Render(line),
			s.valueBig.Render(score),
			"",
		)
	}

	level := s.sectionLbl.Render(fmt.Sprintf("Level: %d", state.Level))
	room := s.sectionLbl.Render(fmt.Sprintf("Room:  %s", m.player.room.id))

	parts := []string{header, "\n", divider, "\n"}
	parts = append(parts, playerRows...)
	parts = append(parts, divider, "\n", level, room)

	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return s.info.Render(body)
}
