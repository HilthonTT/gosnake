package server

import (
	"log"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/muesli/termenv"
)

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
			_, _, active := s.Pty()
			cmds := s.Command()

			if !active || len(cmds) < 1 {
				_, _ = s.Write([]byte(usage("A PTY is required — add the -t flag.")))
				_ = s.Exit(1)
				return
			}

			roomID := cmds[0]
			password := ""
			if len(cmds) > 1 {
				password = cmds[1]
			}

			// Find or create the room.
			room := srv.FindRoom(roomID)
			if room == nil {
				log.Printf("room %q created with password %q", roomID, password)
				room = srv.NewRoom(roomID, password)
			}

			// Password check.
			if room.password != password {
				_, _ = s.Write([]byte(usage("Incorrect room password.")))
				_ = s.Exit(1)
				return
			}

			// Assign player slot.
			p, err := room.AddPlayer(s)
			if err != nil {
				_, _ = s.Write([]byte(err.Error() + "\n"))
				_ = s.Exit(1)
				return
			}

			log.Printf("%s joined room %q [%s]", s.User(), roomID, s.RemoteAddr())
			p.StartGame()
			log.Printf("%s left room %q [%s]", s.User(), roomID, s.RemoteAddr())

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
