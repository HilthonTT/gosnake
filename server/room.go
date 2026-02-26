package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/multi"
)

const (
	idleTimeout = 3 * time.Minute
)

// Room is a single game session. At most MaxPlayers people play; additional
// connections join as observers. The room owns the authoritative game state
// and drives the tick loop via a time.Ticker.
type Room struct {
	id       string
	password string

	mu          sync.RWMutex
	players     map[string]*Player // keyed by public-key string
	playerNames []string           // ordered; used when restarting
	nextIndex   int                // next player slot to assign (0-MaxPlayers)
	started     bool               // true once the first game tick fires

	game *multi.Game // nil until game starts

	sync   chan tea.Msg  // inbound messages from player models
	done   chan struct{} // closed by Close() to stop listen()
	finish chan string   // receives room id when the room should be deleted
}

func newRoom(id, password string, finish chan string) *Room {
	r := &Room{
		id:       id,
		password: password,
		players:  make(map[string]*Player),
		sync:     make(chan tea.Msg, 128),
		done:     make(chan struct{}, 1),
		finish:   finish,
	}
	go r.listen()
	return r
}

func (r *Room) String() string {
	return r.id
}

// Close gracefully shuts down the room: notifies players and signals listen().
func (r *Room) Close() {
	log.Printf("closing room %s", r)

	r.mu.RLock()
	for _, p := range r.players {
		_, _ = p.WriteString("\nServer is shutting down. Goodbye!\n")
		p.closeOnce()
	}
	r.mu.RUnlock()

	// Signal listen() to exit. Use non-blocking send in case Close is called
	// multiple times (e.g. idle timeout then explicit shutdown).
	select {
	case r.done <- struct{}{}:
	default:
	}

	r.finish <- r.id
}

// AddPlayer assigns a session to a player or observer slot and wires it up.
func (r *Room) AddPlayer(s ssh.Session) (*Player, error) {
	log.Println("Calling public key")

	k := s.PublicKey()

	log.Println("After calling public key")
	if k == nil {
		log.Printf("[AddPlayer] No public Key")
		return nil, fmt.Errorf("no public key — re-run with: ssh -i <key> ...")
	}
	pub := PublicKey{key: k}

	r.mu.Lock()

	log.Println("Calling lock")

	if _, ok := r.players[pub.String()]; ok {
		r.mu.Unlock()
		log.Printf("[AddPlayer] Already connected")
		return nil, fmt.Errorf("you are already connected to this room")
	}

	idx := -1
	if !r.started && r.nextIndex < multi.MaxPlayers {
		idx = r.nextIndex
		r.nextIndex++
		r.playerNames = append(r.playerNames, s.User())
	}

	p := &Player{
		room:        r,
		session:     s,
		key:         pub,
		playerIndex: idx,
	}

	log.Println("After indexing")

	p.game = newSharedMultiGame(p, r.sync)

	log.Println("After new shared game")

	ptyInfo, wchan, active := s.Pty()
	log.Printf("[AddPlayer] user=%s active=%v pty=%dx%d playerIndex=%d",
		s.User(), active, ptyInfo.Window.Width, ptyInfo.Window.Height, idx)

	if !active {
		log.Printf("[AddPlayer] WARNING: no active PTY for %s", s.User())
	}
	if ptyInfo.Window.Width == 0 || ptyInfo.Window.Height == 0 {
		log.Printf("[AddPlayer] WARNING: zero dimensions for %s", s.User())
	}

	p.game.width = ptyInfo.Window.Width
	p.game.height = ptyInfo.Window.Height
	p.wchan = wchan

	log.Printf("[AddPlayer] model dimensions set to %dx%d for %s",
		p.game.width, p.game.height, s.User())

	log.Println("Before calling new program")

	prog := tea.NewProgram(
		p.game,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(s),
		tea.WithOutput(s),
	)
	p.program = prog

	log.Println("After calling new program")

	r.players[pub.String()] = p
	log.Printf("[AddPlayer] room %s now has %d player(s)", r.id, len(r.players))

	// Unlock BEFORE broadcasting. p.program.Send() blocks if the Bubble Tea
	// program hasn't called Run() yet; holding the lock here causes a deadlock
	// when a second player connects concurrently and also tries to AddPlayer.
	// joinMsg := NoteMsg(fmt.Sprintf("%s joined as %s", s.User(), p.roleString()))
	r.mu.Unlock()

	log.Println("Unlocked lock")

	// r.broadcast(joinMsg)

	log.Println("After broadcast")

	return p, nil
}

// broadcastLocked sends msg to all players. Caller MUST hold at least r.mu.RLock.
func (r *Room) broadcastLocked(msg tea.Msg) {
	for _, p := range r.players {
		p.Send(msg)
	}
}

// broadcast sends msg to all players (acquires RLock internally).
func (r *Room) broadcast(msg tea.Msg) {
	r.mu.RLock()
	r.broadcastLocked(msg)
	r.mu.RUnlock()
}

// sendNote broadcasts a plain-text note to all players.
func (r *Room) sendNote(s string) { r.broadcast(NoteMsg(s)) }

// broadcastState deep-copies the current game state and sends it to everyone.
// Must NOT be called while holding r.mu.
func (r *Room) broadcastState(died []int) {
	if r.game == nil {
		return
	}

	// Deep-copy the matrix so every player gets its own slice.
	src := r.game.Matrix()
	matrix := make(snake.Matrix, len(src))
	for i, row := range src {
		matrix[i] = make([]byte, len(row))
		copy(matrix[i], row)
	}

	// Snapshot players (value copy — no shared pointers).
	rawPlayers := r.game.Players()
	snapshots := make([]PlayerSnapshot, len(rawPlayers))
	for i, p := range rawPlayers {
		snapshots[i] = PlayerSnapshot{
			Index:  p.Index,
			Name:   p.Name,
			Score:  p.Score,
			Alive:  p.Alive,
			Length: len(p.Points),
		}
	}

	msg := GameStateMsg{
		Matrix:  matrix,
		Players: snapshots,
		Level:   r.game.Level(),
		Over:    r.game.IsOver(),
		Winner:  r.game.Winner(),
		Died:    died,
	}
	r.broadcast(msg)
}

// listen is the room's single-threaded event loop.

func (r *Room) listen() {
	ticker := time.NewTicker(snake.GetTickInterval(1))
	defer ticker.Stop()

	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()

	for {
		select {

		// Shutdown
		case <-r.done:
			return

		// Idle timeout
		case <-idle.C:
			log.Printf("idle timeout for room %s", r)
			r.Close()
			return

		// Inbound player messages
		case msg, ok := <-r.sync:
			if !ok {
				return
			}
			switch m := msg.(type) {

			case NoteMsg:
				r.broadcast(m)

			case DirectionMsg:
				if r.game != nil {
					r.game.ChangeDirection(m.PlayerIndex, multi.Direction(m.Direction))
				}

			case RestartMsg:
				// Any player can trigger an immediate restart.
				r.doRestart()
			}

		//  Game tick
		case <-ticker.C:
			// Start the game once ≥2 players are connected.
			if r.game == nil {
				r.mu.RLock()
				nPlayers := r.nextIndex
				names := make([]string, len(r.playerNames))
				copy(names, r.playerNames)
				r.mu.RUnlock()

				if nPlayers >= 2 {
					r.mu.Lock()
					r.started = true
					r.mu.Unlock()

					r.game = multi.NewGame(names)
					r.sendNote("Game started! Good luck!")
					r.broadcastState(nil)
				}
				continue
			}

			// Don't tick a finished game — wait for a restart vote.
			if r.game.IsOver() {
				continue
			}

			died := r.game.Tick()

			// Adjust tick speed to the new level after the move.
			ticker.Reset(snake.GetTickInterval(r.game.Level()))

			r.broadcastState(died)
		}
	}
}

// doRestart resets the game and notifies all players.
func (r *Room) doRestart() {
	r.mu.RLock()
	names := make([]string, len(r.playerNames))
	copy(names, r.playerNames)
	r.mu.RUnlock()

	r.game = multi.NewGame(names)

	// Tell every client to clear their local state before the first tick arrives.
	r.broadcast(RestartMsg{})
	r.sendNote("Game restarted!")
}
