// Package leaderboard is a client for the Tips.Api online leaderboard.
package leaderboard

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ErrValidation is returned when the API rejects a submission as invalid.
var ErrValidation = errors.New("validation error")

type LeaderboardService struct {
	client       *http.Client // short-lived request/response calls
	streamClient *http.Client // long-lived SSE streams (no overall timeout)
	baseURL      string
}

// Options configures a LeaderboardService.
type Options struct {
	// BaseURL of the Tips.Api server, e.g. "https://localhost:5001".
	BaseURL string

	// InsecureSkipVerify disables TLS verification (local dev certs only).
	InsecureSkipVerify bool

	// Timeout for non-streaming requests. Defaults to 10s.
	Timeout time.Duration
}

func NewLeaderboardService(opts Options) (*LeaderboardService, error) {
	if _, err := url.ParseRequestURI(opts.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid leaderboard API URL %q: %w", opts.BaseURL, err)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in for local dev certs
	}

	return &LeaderboardService{
		client:       &http.Client{Timeout: opts.Timeout, Transport: transport},
		streamClient: &http.Client{Transport: transport},
		baseURL:      strings.TrimRight(opts.BaseURL, "/"),
	}, nil
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, top int) ([]LeaderboardEntry, error) {
	u, err := url.Parse(s.baseURL + "/api/v1/leaderboard")
	if err != nil {
		return nil, fmt.Errorf("build url: %w", err)
	}
	if top > 0 {
		q := u.Query()
		q.Set("top", strconv.Itoa(top))
		u.RawQuery = q.Encode()
	}

	resp, err := s.do(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, unexpectedStatus(resp)
	}

	collection, err := decodeJSON[CollectionResponse[LeaderboardEntry]](resp.Body)
	return collection.Items, err
}

func (s *LeaderboardService) GetByPlayer(ctx context.Context, playerName string) ([]LeaderboardEntry, error) {
	resp, err := s.do(ctx, http.MethodGet, s.baseURL+"/api/v1/leaderboard/player/"+url.PathEscape(playerName), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		collection, err := decodeJSON[CollectionResponse[LeaderboardEntry]](resp.Body)
		return collection.Items, err
	case http.StatusNotFound:
		return []LeaderboardEntry{}, nil
	default:
		return nil, unexpectedStatus(resp)
	}
}

func (s *LeaderboardService) SubmitScore(ctx context.Context, req SubmitScoreRequest) (LeaderboardEntry, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return LeaderboardEntry{}, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := s.do(ctx, http.MethodPost, s.baseURL+"/api/v1/leaderboard", bytes.NewReader(body))
	if err != nil {
		return LeaderboardEntry{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return decodeJSON[LeaderboardEntry](resp.Body)
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		// ValidationProblem — surface the body so the caller can inspect it.
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return LeaderboardEntry{}, fmt.Errorf("%w: %s", ErrValidation, raw)
	default:
		return LeaderboardEntry{}, unexpectedStatus(resp)
	}
}

func (s *LeaderboardService) DeleteEntry(ctx context.Context, entryID string) (bool, error) {
	resp, err := s.do(ctx, http.MethodDelete, s.baseURL+"/api/v1/leaderboard/"+url.PathEscape(entryID), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, unexpectedStatus(resp)
	}
}

// StreamLeaderboard subscribes to live leaderboard changes. Both channels are
// closed when the stream ends; cancel ctx to stop it.
func (s *LeaderboardService) StreamLeaderboard(ctx context.Context, lastEventID string) (<-chan LeaderboardChangeEvent, <-chan error) {
	events := make(chan LeaderboardChangeEvent)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/v1/leaderboard/realtime", nil)
		if err != nil {
			errs <- fmt.Errorf("build request: %w", err)
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		if lastEventID != "" {
			req.Header.Set("Last-Event-ID", lastEventID)
		}

		resp, err := s.streamClient.Do(req)
		if err != nil {
			errs <- fmt.Errorf("connect to SSE stream: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs <- unexpectedStatus(resp)
			return
		}

		if err := parseSSEStream(ctx, resp.Body, events); err != nil && !errors.Is(err, context.Canceled) {
			errs <- err
		}
	}()

	return events, errs
}

func (s *LeaderboardService) do(ctx context.Context, method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	return resp, nil
}

// parseSSEStream reads server-sent events from r and forwards the JSON payload
// of every event to out until r is exhausted or ctx is cancelled.
func parseSSEStream(ctx context.Context, r io.Reader, out chan<- LeaderboardChangeEvent) error {
	scanner := bufio.NewScanner(r)

	var data []string
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))

		case line == "":
			// Blank line = end of event — dispatch if we have data.
			if len(data) == 0 {
				continue
			}
			payload := strings.Join(data, "\n")
			data = data[:0]

			var event LeaderboardChangeEvent
			if err := json.Unmarshal([]byte(payload), &event); err != nil || event.ChangeType == "" {
				// Skip malformed events rather than killing the stream.
				continue
			}

			select {
			case out <- event:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("read SSE stream: %w", err)
	}

	return nil
}

func decodeJSON[T any](r io.Reader) (T, error) {
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}

func unexpectedStatus(resp *http.Response) error {
	return fmt.Errorf("unexpected status %s", resp.Status)
}
