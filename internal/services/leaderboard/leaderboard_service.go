package leaderboard

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/HilthonTT/gosnake/internal/decoder"
)

const (
	BaseURL = "https://localhost:5001"
)

type LeaderboardService struct {
	client  *http.Client
	baseURL string
}

func NewLeaderboardService() *LeaderboardService {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &LeaderboardService{
		client: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		baseURL: BaseURL,
	}
}

func (s *LeaderboardService) GetLeaderboard(ctx context.Context, top int) ([]LeaderboardEntry, error) {
	u, _ := url.Parse(s.baseURL + "/api/v1/leaderboard")
	if top > 0 {
		q := u.Query()
		q.Set("top", strconv.Itoa(top))
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := decoder.DoJSON[CollectionResponse[LeaderboardEntry]](s.client, req)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (s *LeaderboardService) GetByPlayer(ctx context.Context, playerName string) ([]LeaderboardEntry, error) {
	endpoint := s.baseURL + "/api/v1/leaderboard/player/" + url.PathEscape(playerName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []LeaderboardEntry{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	collection, err := decoder.DecodeJSON[CollectionResponse[LeaderboardEntry]](resp.Body)
	if err != nil {
		return nil, err
	}
	return collection.Items, nil
}

func (s *LeaderboardService) SubmitScore(ctx context.Context, req SubmitScoreRequest) (LeaderboardEntry, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return LeaderboardEntry{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.baseURL+"/api/v1/leaderboard",
		bytes.NewReader(body),
	)
	if err != nil {
		return LeaderboardEntry{}, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(httpReq)
	if err != nil {
		return LeaderboardEntry{}, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		// ValidationProblem — surface the body so the caller can inspect it.
		raw, _ := io.ReadAll(resp.Body)
		return LeaderboardEntry{}, fmt.Errorf("validation error: %s", raw)
	}
	if resp.StatusCode != http.StatusCreated {
		return LeaderboardEntry{}, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return decoder.DecodeJSON[LeaderboardEntry](resp.Body)
}

func (s *LeaderboardService) DeleteEntry(ctx context.Context, entryID string) (bool, error) {
	endpoint := s.baseURL + "/api/v1/leaderboard/" + url.PathEscape(entryID)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("unexpected status %s", resp.Status)
	}
}

func (s *LeaderboardService) StreamLeaderboard(ctx context.Context, lastEventID string) (<-chan LeaderboardChangeEvent, <-chan error) {
	events := make(chan LeaderboardChangeEvent)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		// Use a client without a timeout — SSE streams are long-lived.

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/api/v1/leaderboard/realtime", nil)
		if err != nil {
			errs <- fmt.Errorf("build request: %w", err)
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		if lastEventID != "" {
			req.Header.Set("Last-Event-ID", lastEventID)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			errs <- fmt.Errorf("connect to SSE stream: %w", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			errs <- fmt.Errorf("unexpected status %s", resp.Status)
			return
		}

		if err := parseSseStream(ctx, resp.Body, events); err != nil {
			errs <- err
		}
	}()

	return events, errs
}

func parseSseStream(ctx context.Context, r io.Reader, out chan<- LeaderboardChangeEvent) error {
	scanner := bufio.NewScanner(r)

	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))

		case line == "":
			// Blank line = end of event — dispatch if we have data.
			if len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				dataLines = dataLines[:0]

				var event LeaderboardChangeEvent
				if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &event); err != nil {
					// Skip malformed events rather than killing the stream.
					continue
				}
				out <- event
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read SSE stream: %w", err)
	}

	return nil
}
