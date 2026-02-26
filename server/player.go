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
	wchan       <-chan ssh.Window
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
	log.Printf("[StartGame] %s starting program (index=%d)", p.session.User(), p.playerIndex)
	errc := make(chan error, 1)

	go func() {
		for {
			select {
			case w, ok := <-p.wchan:
				if !ok {
					log.Printf("[StartGame] wchan closed for %s", p.session.User())
					return
				}
				log.Printf("[StartGame] resize event for %s: %dx%d", p.session.User(), w.Width, w.Height)
				if p.program != nil {
					p.program.Send(tea.WindowSizeMsg{Width: w.Width, Height: w.Height})
				}
			case err := <-errc:
				if err != nil {
					log.Printf("[StartGame] program error for %s: %v", p.session.User(), err)
				}
				return
			case <-p.session.Context().Done():
				log.Printf("[StartGame] context done for %s", p.session.User())
				p.closeOnce()
				return
			}
		}
	}()

	defer func() {
		log.Printf("[StartGame] %s deferred cleanup", p.session.User())
		select {
		case p.room.sync <- NoteMsg(fmt.Sprintf("%s left the room", p)):
		default:
		}
		p.closeOnce()
	}()

	log.Printf("[StartGame] calling Run() for %s", p.session.User())
	_, err := p.program.Run()
	log.Printf("[StartGame] Run() returned for %s: err=%v", p.session.User(), err)
	errc <- err
}
