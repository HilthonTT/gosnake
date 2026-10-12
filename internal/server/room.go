package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
	"charm.land/wish/v2/bubbletea"

	"github.com/HilthonTT/gosnake/pkg/snake"
	"github.com/HilthonTT/gosnake/pkg/snake/modes/multi"
)

// idleTimeout closes a room after this long without player input or a
// running game.
const idleTimeout = 3 * time.Minute

// Room is a single game session. At most MaxPlayers people play; additional
// connections join as observers. The room owns the authoritative game state
// and drives the tick loop via a time.Ticker.
type Room struct {
	id       string
	password string

	mu      sync.RWMutex
	players map[string]*Player // keyed by public-key string
	started bool               // true once the first game has begun

	game *multi.Game // owned by listen(); nil until the game starts

	sync      chan tea.Msg  // inbound messages from player models
	done      chan struct{} // closed by Close() to stop listen()
	closeOnce sync.Once
	onClose   func(id string)
}

func newRoom(id, password string, onClose func(id string)) *Room {
	r := &Room{
		id:       id,
		password: password,
		players:  make(map[string]*Player),
		sync:     make(chan tea.Msg, 128),
		done:     make(chan struct{}),
		onClose:  onClose,
	}
	go r.listen()
	return r
}

func (r *Room) String() string {
	return r.id
}

// CheckPassword reports whether password matches the room's, in constant time.
func (r *Room) CheckPassword(password string) bool {
	return subtle.ConstantTimeCompare([]byte(r.password), []byte(password)) == 1
}

// Close shuts the room down: disconnects players, stops listen() and
// unregisters the room. Safe to call more than once and from any goroutine.
func (r *Room) Close() {
	r.closeOnce.Do(func() {
		slog.Info("closing room", "room", r.id)

		close(r.done)

		for _, p := range r.snapshotPlayers() {
			_, _ = p.WriteString("\nThe room has closed. Goodbye!\n")
			p.closeOnce()
		}

		if r.onClose != nil {
			r.onClose(r.id)
		}
	})
}

var errAlreadyConnected = errors.New("you are already connected to this room")

// AddPlayer assigns a session to a player or observer slot and wires it up.
func (r *Room) AddPlayer(s ssh.Session) (*Player, error) {
	k := s.PublicKey()
	if k == nil {
		return nil, errors.New("no public key: re-run with ssh -i <key>")
	}
	pub := PublicKey{key: k}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.players[pub.String()]; ok {
		return nil, errAlreadyConnected
	}

	p := &Player{room: r, session: s, key: pub}
	p.setIndex(-1)
	if !r.started && r.activePlayersLocked() < multi.MaxPlayers {
		p.setIndex(r.nextFreeIndexLocked())
	}

	p.game = newSharedMultiGame(p, r.sync)

	ptyInfo, wchan, _ := s.Pty()
	p.game.width = ptyInfo.Window.Width
	p.game.height = ptyInfo.Window.Height
	p.wchan = wchan

	// MakeOptions wires the program to the session's input/output and
	// detects the client's color profile.
	opts := append(bubbletea.MakeOptions(s), tea.WithWindowSize(ptyInfo.Window.Width, ptyInfo.Window.Height))
	p.program = tea.NewProgram(p.game, opts...)

	r.players[pub.String()] = p

	// Routed through listen() so the note is delivered once every program
	// is actually running.
	r.notify(NoteMsg(fmt.Sprintf("%s joined as %s", s.User(), p.roleString())))

	return p, nil
}

// removePlayer drops p from the room. Called by Player.closeOnce.
func (r *Room) removePlayer(p *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if cur, ok := r.players[p.key.String()]; ok && cur == p {
		delete(r.players, p.key.String())
	}
}

// notify queues msg for broadcast without blocking.
func (r *Room) notify(msg tea.Msg) {
	select {
	case r.sync <- msg:
	default:
	}
}

// snapshotPlayers returns the current players so callers can act on them
// without holding the lock (Program.Send and closeOnce may block or re-lock).
func (r *Room) snapshotPlayers() []*Player {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]*Player, 0, len(r.players))
	for _, p := range r.players {
		out = append(out, p)
	}
	return out
}

// broadcast sends msg to all players.
func (r *Room) broadcast(msg tea.Msg) {
	for _, p := range r.snapshotPlayers() {
		p.Send(msg)
	}
}

// broadcastState deep-copies the current game state and sends it to everyone.
func (r *Room) broadcastState(died []int) {
	if r.game == nil {
		return
	}

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

	r.broadcast(GameStateMsg{
		Matrix:  r.game.Matrix().Clone(),
		Players: snapshots,
		Level:   r.game.Level(),
		Over:    r.game.IsOver(),
		Winner:  r.game.Winner(),
		Died:    died,
	})
}

// listen is the room's single-threaded event loop.
// It owns the game object exclusively — no other goroutine touches r.game.
func (r *Room) listen() {
	ticker := time.NewTicker(snake.GetTickInterval(1))
	defer ticker.Stop()

	idle := time.NewTimer(idleTimeout)
	defer idle.Stop()
	resetIdle := func() { idle.Reset(idleTimeout) }

	for {
		select {
		case <-r.done:
			return

		case <-idle.C:
			slog.Info("idle timeout", "room", r.id)
			go r.Close() // Close waits on nothing from listen, but keep it off this goroutine.
			return

		case msg := <-r.sync:
			switch m := msg.(type) {
			case NoteMsg:
				r.broadcast(m)

			case DirectionMsg:
				resetIdle()
				if r.game != nil {
					r.game.ChangeDirection(m.PlayerIndex, snake.Direction(m.Direction))
				}

			case RestartMsg:
				// Any player can trigger an immediate restart.
				resetIdle()
				r.startGame("Game restarted!")
			}

		case <-ticker.C:
			if r.game == nil {
				// Start the game once ≥2 players are connected.
				if len(r.activePlayerNames()) >= 2 {
					r.startGame("Game started! Good luck!")
					ticker.Reset(snake.GetTickInterval(r.game.Level()))
				}
				continue
			}

			// Don't tick a finished game — wait for a restart.
			if r.game.IsOver() {
				continue
			}

			resetIdle()
			died := r.game.Tick()

			// Adjust tick speed to the new level after the move.
			ticker.Reset(snake.GetTickInterval(r.game.Level()))

			r.broadcastState(died)
		}
	}
}

// startGame (re)creates the game for every connected player slot. Slots are
// compacted first so players who left before the start don't leave a ghost
// snake behind.
func (r *Room) startGame(note string) {
	names := r.compactSlots()
	if len(names) < 2 {
		r.broadcast(NoteMsg("Waiting for more players..."))
		return
	}

	r.game = multi.NewGame(names)

	// Tell every client to clear their local state before the first tick arrives.
	r.broadcast(RestartMsg{})
	r.broadcast(NoteMsg(note))
	r.broadcastState(nil)
}

// compactSlots renumbers connected players 0..n-1 (keeping their order),
// marks the room as started, and returns their names in slot order.
func (r *Room) compactSlots() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	active := r.activeLocked()
	names := make([]string, len(active))
	for i, p := range active {
		p.setIndex(i)
		names[i] = p.session.User()
	}
	r.started = true
	return names
}

// activePlayerNames returns the names of players holding a slot.
func (r *Room) activePlayerNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := r.activeLocked()
	names := make([]string, len(active))
	for i, p := range active {
		names[i] = p.session.User()
	}
	return names
}

// activeLocked returns slot holders sorted by slot. Caller holds r.mu.
func (r *Room) activeLocked() []*Player {
	var active []*Player
	for _, p := range r.players {
		if p.Index() >= 0 {
			active = append(active, p)
		}
	}
	slices.SortFunc(active, func(a, b *Player) int { return a.Index() - b.Index() })
	return active
}

func (r *Room) activePlayersLocked() int {
	return len(r.activeLocked())
}

// nextFreeIndexLocked returns the lowest unused slot. Caller holds r.mu.
func (r *Room) nextFreeIndexLocked() int {
	used := make(map[int]bool, len(r.players))
	for _, p := range r.players {
		used[p.Index()] = true
	}
	for i := range multi.MaxPlayers {
		if !used[i] {
			return i
		}
	}
	return -1
}
