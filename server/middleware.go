package server

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/muesli/termenv"
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
				log.Printf("rate limit exceeded for IP %s — closing connection", ip)
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
		lipgloss.SetColorProfile(termenv.ANSI256)

		return func(s ssh.Session) {
			log.Printf("Hit 1")

			_, _, active := s.Pty()
			cmds := s.Command()

			if !active || len(cmds) < 1 {
				log.Printf("Hit 2")
				_, _ = s.Write([]byte(usage("A PTY is required — add the -t flag.")))
				_ = s.Exit(1)
				return
			}

			log.Printf("Hit 3")

			roomID := cmds[0]
			password := ""
			if len(cmds) > 1 {
				password = cmds[1]
			}

			log.Printf("Hit 4")

			// Find or create the room.
			room := srv.FindRoom(roomID)
			if room == nil {
				log.Printf("Hit 5")
				log.Printf("room %q created with password %q", roomID, password)
				room = srv.NewRoom(roomID, password)
			}

			// Password check.
			if room.password != password {
				log.Printf("Hit 6")
				_, _ = s.Write([]byte(usage("Incorrect room password.")))
				_ = s.Exit(1)
				return
			}

			log.Printf("Hit 7")

			// Assign player slot.
			p, err := room.AddPlayer(s)
			if err != nil {
				log.Printf("Hit 8")
				_, _ = s.Write([]byte(err.Error() + "\n"))
				_ = s.Exit(1)
				return
			}

			log.Printf("Hit 9")

			log.Printf("%s joined room %q [%s]", s.User(), roomID, s.RemoteAddr())
			p.StartGame()
			log.Printf("%s left room %q [%s]", s.User(), roomID, s.RemoteAddr())

			log.Printf("Hit 10")

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
