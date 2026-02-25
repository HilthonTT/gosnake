package server

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/multi"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
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

func (r *Room) Close() {
	log.Printf("closing room %s", r)

	r.mu.RLock()
	for _, p := range r.players {
		_, _ = p.WriteString("\nServer is shutting down. Goodbye!\n")
		p.closeOnce()
	}
	r.mu.RUnlock()

	select {
	case r.done <- struct{}{}:
	default:
	}

	r.finish <- r.id
}

func (r *Room) AddPlayer(s ssh.Session) (*Player, error) {
	k := s.PublicKey()
	if k == nil {
		return nil, fmt.Errorf("no public key — re-run with: ssh -i <key> ...")
	}
	pub := PublicKey{key: k}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[pub.String()]; ok {
		return nil, fmt.Errorf("you are already connected to this room")
	}

	// Assign player index or observer.
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

	p.game = newSharedMultiGame(p, r.sync)

	prog := tea.NewProgram(
		p.game,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(s),
		tea.WithOutput(s),
	)
	p.program = prog

	r.players[pub.String()] = p

	// Notify all existing players that someone new arrived.
	r.broadcastLocked(NoteMsg(fmt.Sprintf("%s joined as %s", s.User(), p.roleString())))

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
func (r *Room) sendNote(s string) {
	r.broadcast(NoteMsg(s))
}

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

	// Snapshot players (value-copy - no shared pointers)
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
// It owns the game object exclusively — no other goroutine touches r.game.
func (r *Room) listen() {
	ticker := time.NewTicker(snake.GetTickInterval(1))
	defer ticker.Stop()

	for {
		select {

		// Shutdown
		case <-r.done:
			return

		// Idle timeout
		case <-time.After(idleTimeout):
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

		// Game tick
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
