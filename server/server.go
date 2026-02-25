package server

import (
	"context"
	"fmt"
	"log"
	"sync"

	gossh "golang.org/x/crypto/ssh"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// PublicKey wraps ssh.PublicKey and provides a stable string key for maps.
type PublicKey struct {
	key ssh.PublicKey
}

// String returns the authorized-keys representation of the public key,
// which is unique and stable across sessions.
func (pk PublicKey) String() string {
	return fmt.Sprintf("%s", gossh.MarshalAuthorizedKey(pk.key))
}

// Server manages multiplayer snake rooms over SSH.
type Server struct {
	host  string
	port  int
	srv   *ssh.Server
	rooms map[string]*Room
	mu    sync.Mutex
}

// NewServer creates and configures the SSH server.
// keyPath is the path used to persist the server's host key across restarts.
func NewServer(keyPath, host string, port int) (*Server, error) {
	s := &Server{
		host:  host,
		port:  port,
		rooms: make(map[string]*Room),
	}

	ws, err := wish.NewServer(
		// Accept any password and any public key — room passwords handle auth.
		ssh.PasswordAuth(func(_ ssh.Context, _ string) bool { return true }),
		ssh.PublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool { return true }),
		wish.WithHostKeyPath(keyPath),
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithMiddleware(multiMiddleware(s)),
	)

	if err != nil {
		return nil, err
	}

	s.srv = ws
	return s, nil
}

// Start begins serving SSH connections. It blocks until the server stops.
func (s *Server) Start() error {
	return s.srv.ListenAndServe()
}

// Shutdown gracefully closes all rooms and the underlying SSH server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	for _, room := range s.rooms {
		room.Close()
	}
	s.mu.Unlock()
	return s.srv.Shutdown(ctx)
}

// FindRoom returns the room with the given id, or nil if it doesn't exist.
func (s *Server) FindRoom(id string) *Room {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rooms[id]
}

// NewRoom creates, registers, and returns a new room.
// A goroutine watches the finish channel so the room self-removes on close
func (s *Server) NewRoom(id, password string) *Room {
	finish := make(chan string, 1)
	go func() {
		rid := <-finish
		log.Printf("deleting room %s", rid)
		s.mu.Lock()
		delete(s.rooms, rid)
		s.mu.Unlock()
		close(finish)
	}()

	room := newRoom(id, password, finish)
	s.mu.Lock()
	s.rooms[id] = room
	s.mu.Unlock()

	return room
}
