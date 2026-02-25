package server

import (
	"fmt"
	"log"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
)

// Player represents one connected SSH session inside a room.
// playerIndex == -1 means the session is an observer.
type Player struct {
	room        *Room
	session     ssh.Session
	program     *tea.Program
	game        *SharedMultiGame
	playerIndex int
	key         PublicKey
	once        sync.Once
}

// String returns a human-readable label used in log messages and notes.
func (p *Player) String() string {
	return fmt.Sprintf("%s (%s)", p.session.User(), p.roleString())
}

func (p *Player) roleString() string {
	if p.playerIndex >= 0 {
		return fmt.Sprintf("Player %d", p.playerIndex+1)
	}
	return "Observer"
}

// Send dispatches a Bubble Tea message to this player's program.
func (p *Player) Send(msg tea.Msg) {
	if p.program != nil {
		p.program.Send(msg)
	}
}

// Write writes raw bytes to the underlying SSH session.
func (p *Player) Write(b []byte) (int, error) {
	return p.session.Write(b)
}

// WriteString writes a string to the underlying SSH session.
func (p *Player) WriteString(s string) (int, error) {
	return p.session.Write([]byte(s))
}

// closeOnce tears down this player's program and removes them from the room.
// It is safe to call from multiple goroutines; only the first call has effect.
func (p *Player) closeOnce() {
	p.once.Do(func() {
		if p.program != nil {
			p.program.Kill()
		}
		p.session.Close()

		p.room.mu.Lock()
		delete(p.room.players, p.key.String())
		p.room.mu.Unlock()
	})
}

// StartGame runs the Bubble Tea program for this player, blocking until the
// session ends.  It also forwards terminal resize events and handles context
// cancellation (e.g. the client disconnecting mid-game).
func (p *Player) StartGame() {
	_, wchan, _ := p.session.Pty()
	errc := make(chan error, 1)

	// Resize / disconnect watcher.
	go func() {
		select {
		case w := <-wchan:
			if p.program != nil {
				p.program.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
			}
		case err := <-errc:
			if err != nil {
				log.Printf("program error for %s: %v", p, err)
			}
		case <-p.session.Context().Done():
			p.closeOnce()
		}
	}()

	defer func() {
		// Let the room know this player left so their name shows as disconnected.
		select {
		case p.room.sync <- NoteMsg(fmt.Sprintf("%s left the room", p)):
		default:
		}
		p.closeOnce()
	}()

	_, err := p.program.Run()
	errc <- err
}
