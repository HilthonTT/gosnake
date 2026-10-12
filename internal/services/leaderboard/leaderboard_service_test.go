package leaderboard

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestService(t *testing.T, h http.HandlerFunc) *LeaderboardService {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	s, err := NewLeaderboardService(Options{BaseURL: srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewLeaderboardServiceRejectsBadURL(t *testing.T) {
	if _, err := NewLeaderboardService(Options{BaseURL: "not a url"}); err == nil {
		t.Fatal("expected an error")
	}
}

func TestSubmitScore(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/leaderboard" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var req SubmitScoreRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(LeaderboardEntry{EntryID: "abc", PlayerName: req.PlayerName, Score: req.Score})
	})

	entry, err := s.SubmitScore(context.Background(), SubmitScoreRequest{PlayerName: "neo", Score: 42, Level: 1, SnakeLength: 3})
	if err != nil {
		t.Fatal(err)
	}
	if entry.EntryID != "abc" || entry.PlayerName != "neo" || entry.Score != 42 {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestSubmitScoreValidationError(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"errors":{"playername":["required"]}}`, http.StatusBadRequest)
	})

	_, err := s.SubmitScore(context.Background(), SubmitScoreRequest{})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}

func TestGetByPlayerNotFoundIsEmpty(t *testing.T) {
	s := newTestService(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	entries, err := s.GetByPlayer(context.Background(), "nobody")
	if err != nil || len(entries) != 0 {
		t.Fatalf("entries = %v, err = %v", entries, err)
	}
}

func TestParseSSEStream(t *testing.T) {
	stream := strings.Join([]string{
		"retry: 5000",
		"",
		"event: leaderboard-change",
		"id: 0",
		`data: {"changeType":"EntryAdded","entry":{"entryId":"a","score":10}}`,
		"",
		"data: not json",
		"",
		"data: null",
		"",
		"id: 1",
		`data: {"changeType":"EntryDeleted",`,
		`data: "entry":{"entryId":"a"}}`,
		"", // terminates the last event; the trailing join adds the newline
		"",
	}, "\n")

	out := make(chan LeaderboardChangeEvent, 10)
	if err := parseSSEStream(context.Background(), strings.NewReader(stream), out); err != nil {
		t.Fatal(err)
	}
	close(out)

	var got []LeaderboardChangeEvent
	for e := range out {
		got = append(got, e)
	}
	if len(got) != 2 || got[0].ChangeType != EntryAdded || got[1].ChangeType != EntryDeleted || got[0].Entry.Score != 10 {
		t.Fatalf("events = %+v", got)
	}
}

func TestParseSSEStreamStopsOnCancel(t *testing.T) {
	stream := `data: {"changeType":"EntryAdded","entry":{}}` + "\n\n"
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() { done <- parseSSEStream(ctx, strings.NewReader(stream), make(chan LeaderboardChangeEvent)) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("parser blocked on an unread channel after cancellation")
	}
}
