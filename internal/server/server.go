// Package server hosts multiplayer snake rooms over SSH.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	gossh "golang.org/x/crypto/ssh"

	"charm.land/ssh"
	"charm.land/wish/v2"
	"charm.land/wish/v2/activeterm"
	"charm.land/wish/v2/logging"
	"charm.land/wish/v2/ratelimiter"
	"charm.land/wish/v2/recover"
)

// ErrServerClosed is returned by Start after Shutdown.
var ErrServerClosed = ssh.ErrServerClosed

// PublicKey wraps ssh.PublicKey and provides a stable string key for maps.
type PublicKey struct {
	key ssh.PublicKey
}

// String returns the authorized-keys representation of the public key,
// which is unique and stable across sessions.
func (pk PublicKey) String() string {
	return string(gossh.MarshalAuthorizedKey(pk.key))
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

	globalRL := ratelimiter.NewRateLimiter(globalRateLimit, globalBurst, globalMaxSessions)
	ipRL := newIPRateLimiter()

	ws, err := wish.NewServer(
		// Accept any password and any public key — room passwords handle auth.
		ssh.PasswordAuth(func(_ ssh.Context, _ string) bool { return true }),
		ssh.PublicKeyAuth(func(_ ssh.Context, _ ssh.PublicKey) bool { return true }),
		wish.WithHostKeyPath(keyPath),
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		// Middlewares run last-to-first: rate limits, then recover/logging,
		// then the PTY check, then the game itself.
		wish.WithMiddleware(
			multiMiddleware(s),
			activeterm.Middleware(),
			logging.Middleware(),
			recover.Middleware(),
			perIPMiddleware(ipRL),
			ratelimiter.Middleware(globalRL),
		),
	)
	if err != nil {
		return nil, err
	}

	s.srv = ws
	return s, nil
}

// Start begins serving SSH connections. It blocks until the server stops.
func (s *Server) Start() error {
	slog.Info("SSH server listening", "host", s.host, "port", s.port)
	return s.srv.ListenAndServe()
}

// Shutdown gracefully closes all rooms and the underlying SSH server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	rooms := make([]*Room, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}
	s.mu.Unlock()

	// Close outside the lock: Room.Close calls back into removeRoom.
	for _, room := range rooms {
		room.Close()
	}
	return s.srv.Shutdown(ctx)
}

// FindOrCreateRoom returns the room with the given id, creating it with
// password if it doesn't exist. created reports whether a new room was made.
func (s *Server) FindOrCreateRoom(id, password string) (room *Room, created bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if room, ok := s.rooms[id]; ok {
		return room, false
	}

	room = newRoom(id, password, s.removeRoom)
	s.rooms[id] = room
	return room, true
}

// removeRoom unregisters a closed room.
func (s *Server) removeRoom(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("deleting room", "room", id)
	delete(s.rooms, id)
}
