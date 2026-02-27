package leaderboard

import "time"

type SubmitScoreRequest struct {
	PlayerName  string `json:"playerName"`
	Score       int    `json:"score"`
	Level       int    `json:"level"`
	SnakeLength int    `json:"snakeLength"`
}

type LeaderboardEntry struct {
	EntryID     string    `json:"entryId"`
	Score       int       `json:"score"`
	Level       int       `json:"level"`
	SnakeLength int       `json:"snakeLength"`
	PlayedAt    time.Time `json:"playedAt"`
}

type LeaderboardChangeType string

const (
	EntryAdded   LeaderboardChangeType = "EntryAdded"
	EntryDeleted LeaderboardChangeType = "EntryDeleted"
)

type LeaderboardChangeEvent struct {
	ChangeType LeaderboardChangeType `json:"changeType"`
	Entry      LeaderboardEntry      `json:"entry"`
}

type CollectionResponse[T any] struct {
	Items []T `json:"items"`
}
