package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

// syncBuffer is a goroutine-safe bytes.Buffer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func startServer(t *testing.T) (*Server, string) {
	t.Helper()
	port := freePort(t)
	srv, err := NewServer(filepath.Join(t.TempDir(), "hostkey"), "127.0.0.1", port)
	if err != nil {
		t.Fatal(err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Start() }()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(5 * time.Second)
	for {
		conn, err := net.Dial("tcp", addr)
		if err == nil {
			conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not start: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		if err := <-errCh; err != nil && !errors.Is(err, ErrServerClosed) {
			t.Errorf("server stopped with: %v", err)
		}
	})

	return srv, addr
}

// join connects user to room with a 160x60 PTY and returns its output.
func join(t *testing.T, addr, user, room string) (*syncBuffer, *gossh.Session) {
	t.Helper()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}

	client, err := gossh.Dial("tcp", addr, &gossh.ClientConfig{
		User:            user,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec // test server
		Timeout:         5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.RequestPty("xterm-256color", 60, 160, gossh.TerminalModes{}); err != nil {
		t.Fatal(err)
	}

	out := &syncBuffer{}
	session.Stdout = out
	session.Stderr = out
	if _, err := session.StdinPipe(); err != nil {
		t.Fatal(err)
	}
	if err := session.Start(room); err != nil {
		t.Fatal(err)
	}
	return out, session
}

func waitFor(t *testing.T, out *syncBuffer, want ...string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		s := out.String()
		ok := true
		for _, w := range want {
			if !strings.Contains(s, w) {
				ok = false
				break
			}
		}
		if ok {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("output never contained %q; got %d bytes", want, len(out.String()))
}

func TestMultiplayerGameStartsAndShutsDownCleanly(t *testing.T) {
	srv, addr := startServer(t)

	alice, _ := join(t, addr, "alice", "room1")
	waitFor(t, alice, "Waiting")

	bob, _ := join(t, addr, "bob", "room1")
	waitFor(t, bob, "MULTI", "alice", "bob")
	waitFor(t, alice, "MULTI", "bob")

	// Shutting down with players connected used to deadlock.
	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		done <- srv.Shutdown(ctx)
	}()

	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("Shutdown deadlocked with connected players")
	}

	srv.mu.Lock()
	rooms := len(srv.rooms)
	srv.mu.Unlock()
	if rooms != 0 {
		t.Fatalf("rooms left after shutdown: %d", rooms)
	}
}

func TestWrongRoomPasswordIsRejected(t *testing.T) {
	_, addr := startServer(t)

	owner, _ := join(t, addr, "owner", "vault secret")
	waitFor(t, owner, "Waiting")

	intruder, session := join(t, addr, "intruder", "vault wrong")
	_ = session.Wait()
	waitFor(t, intruder, "Incorrect room password")
}

func TestRoomCloseIsIdempotent(t *testing.T) {
	closed := 0
	r := newRoom("r", "", func(string) { closed++ })

	r.Close()
	r.Close()

	if closed != 1 {
		t.Fatalf("onClose called %d times, want 1", closed)
	}
}
