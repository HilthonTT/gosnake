package server

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"charm.land/ssh"
)

// Player represents one connected SSH session inside a room.
// An index of -1 means the session is an observer.
type Player struct {
	room    *Room
	session ssh.Session
	program *tea.Program
	game    *SharedMultiGame
	key     PublicKey
	once    sync.Once
	wchan   <-chan ssh.Window

	// index is read by the player's UI goroutine and rewritten by the room
	// when slots are compacted at game start, hence atomic.
	index atomic.Int64
}

// Index returns the player's slot, or -1 for observers.
func (p *Player) Index() int { return int(p.index.Load()) }

func (p *Player) setIndex(i int) { p.index.Store(int64(i)) }

// String returns a human-readable label used in log messages and notes.
func (p *Player) String() string {
	return fmt.Sprintf("%s (%s)", p.session.User(), p.roleString())
}

func (p *Player) roleString() string {
	if idx := p.Index(); idx >= 0 {
		return fmt.Sprintf("Player %d", idx+1)
	}
	return "Observer"
}

// Send dispatches a Bubble Tea message to this player's program.
func (p *Player) Send(msg tea.Msg) {
	if p.program != nil {
		p.program.Send(msg)
	}
}

// WriteString writes a string to the underlying SSH session.
func (p *Player) WriteString(s string) (int, error) {
	return p.session.Write([]byte(s))
}

// closeOnce tears down this player's program and removes them from the room.
// It is safe to call from multiple goroutines; only the first call has effect.
// The caller must NOT hold room.mu.
func (p *Player) closeOnce() {
	p.once.Do(func() {
		if p.program != nil {
			p.program.Kill()
		}
		_ = p.session.Close()
		p.room.removePlayer(p)
	})
}

// StartGame runs the Bubble Tea program for this player, blocking until the
// session ends. It also forwards terminal resize events and handles context
// cancellation (e.g. the client disconnecting mid-game).
func (p *Player) StartGame() {
	done := make(chan struct{})

	go func() {
		for {
			select {
			case w, ok := <-p.wchan:
				if !ok {
					return
				}
				p.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
			case <-done:
				return
			case <-p.session.Context().Done():
				p.closeOnce()
				return
			}
		}
	}()

	defer func() {
		close(done)
		p.room.notify(NoteMsg(fmt.Sprintf("%s left the room", p)))
		p.closeOnce()
	}()

	if _, err := p.program.Run(); err != nil {
		slog.Debug("player program ended", "user", p.session.User(), "err", err)
	}
}
