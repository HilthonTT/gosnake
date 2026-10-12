package server

import (
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"charm.land/ssh"
	"charm.land/wish/v2"
	"golang.org/x/time/rate"
)

const (
	// ipRateLimit is the maximum sustained connection rate allowed per IP.
	ipRateLimit = rate.Limit(2) // 2 connections per second
	// ipBurst is the number of connections a single IP can make in a burst.
	ipBurst = 5
	// ipLimiterTTL is how long an idle IP limiter is kept before being pruned.
	ipLimiterTTL = 5 * time.Minute
	// globalRateLimit is the maximum sustained connection rate across all IPs.
	globalRateLimit = rate.Limit(10) // 10 connections per second
	// globalBurst is the global burst allowance.
	globalBurst = 20
	// globalMaxSessions is the maximum number of concurrent SSH sessions.
	globalMaxSessions = 100
)

type ipEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*ipEntry
}

func newIPRateLimiter() *ipRateLimiter {
	rl := &ipRateLimiter{
		entries: make(map[string]*ipEntry),
	}
	go rl.pruneLoop()
	return rl
}

func (rl *ipRateLimiter) limiterFor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	e, ok := rl.entries[ip]
	if !ok {
		e = &ipEntry{limiter: rate.NewLimiter(ipRateLimit, ipBurst)}
		rl.entries[ip] = e
	}
	e.lastSeen = time.Now()
	return e.limiter
}

func (rl *ipRateLimiter) pruneLoop() {
	ticker := time.NewTicker(ipLimiterTTL / 2)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-ipLimiterTTL)
		for ip, e := range rl.entries {
			if e.lastSeen.Before(cutoff) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func perIPMiddleware(rl *ipRateLimiter) wish.Middleware {
	return func(next ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			remoteAddr := s.RemoteAddr().String()
			ip, _, err := net.SplitHostPort(remoteAddr)
			if err != nil {
				ip = remoteAddr
			}

			if !rl.limiterFor(ip).Allow() {
				slog.Warn("rate limit exceeded — closing connection", "ip", ip)
				_ = s.Exit(1)
				return
			}

			next(s)
		}
	}
}

// multiMiddleware is the wish middleware that handles all incoming SSH sessions.
// It parses the command arguments to find a room ID and optional password,
// then assigns the session to the appropriate room.
//
// Connection format:
//
//	ssh <name>@<host> -p <port> -t <room-id> [room-password]
func multiMiddleware(srv *Server) wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			_, _, active := s.Pty()
			cmds := s.Command()

			if !active || len(cmds) < 1 {
				_, _ = s.Write([]byte(usage("A PTY and a room id are required — add -t <room-id>.")))
				_ = s.Exit(1)
				return
			}

			roomID := cmds[0]
			password := ""
			if len(cmds) > 1 {
				password = cmds[1]
			}

			room, created := srv.FindOrCreateRoom(roomID, password)
			if created {
				slog.Info("room created", "room", roomID, "protected", password != "")
			}

			if !room.CheckPassword(password) {
				_, _ = s.Write([]byte(usage("Incorrect room password.")))
				_ = s.Exit(1)
				return
			}

			p, err := room.AddPlayer(s)
			if err != nil {
				_, _ = s.Write([]byte(err.Error() + "\n"))
				_ = s.Exit(1)
				return
			}

			slog.Info("player joined", "user", s.User(), "room", roomID, "addr", s.RemoteAddr().String())
			p.StartGame()
			slog.Info("player left", "user", s.User(), "room", roomID, "addr", s.RemoteAddr().String())

			sh(s)
		}
	}
}

func usage(reason string) string {
	lines := []string{
		"GoSnake Multiplayer",
		"",
		"Usage:",
		"  ssh <name>@<host> -p <port> -t <room-id> [room-password]",
		"",
		"Notes:",
		"  • Up to 3 players per room; extras join as observers.",
		"  • The first player to create a room sets its password.",
		"  • The game starts automatically once 2+ players have joined.",
		"",
	}
	if reason != "" {
		lines = append(lines, "Error: "+reason, "")
	}
	return strings.Join(lines, "\n")
}
