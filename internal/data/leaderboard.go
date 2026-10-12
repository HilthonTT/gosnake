package data

import (
	"database/sql"
	"fmt"
)

// GameMode identifies which ruleset was played.
type GameMode string

const (
	GameModeNormal GameMode = "normal"
	GameModeCrazy  GameMode = "crazy"
	// GameModeAI keeps its historical upper-case value so existing rows still
	// group together.
	GameModeAI GameMode = "AI"
)

type LeaderboardEntry struct {
	ID        int
	Name      string
	Score     int
	Level     int
	Mode      GameMode
	CreatedAt string
}

type LeaderboardRepository struct {
	db *sql.DB
}

func NewLeaderboardRepository(db *sql.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db}
}

const selectEntries = `SELECT id, name, score, level, mode, created_at FROM leaderboard`

func (r *LeaderboardRepository) Save(name string, score, level int, mode GameMode) (int, error) {
	const query = `
		INSERT INTO leaderboard (name, score, level, mode)
		VALUES (?, ?, ?, ?)
	`
	res, err := r.db.Exec(query, name, score, level, mode)
	if err != nil {
		return 0, fmt.Errorf("failed to save leaderboard entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve last insert id: %w", err)
	}

	return int(id), nil
}

// All returns every entry, best score first.
func (r *LeaderboardRepository) All() ([]LeaderboardEntry, error) {
	return r.query(selectEntries + ` ORDER BY score DESC`)
}

// GetTopN returns the n best entries.
func (r *LeaderboardRepository) GetTopN(n int) ([]LeaderboardEntry, error) {
	return r.query(selectEntries+` ORDER BY score DESC LIMIT ?`, n)
}

// GetTopNByMode returns the n best entries for a single game mode.
func (r *LeaderboardRepository) GetTopNByMode(n int, mode GameMode) ([]LeaderboardEntry, error) {
	return r.query(selectEntries+` WHERE mode = ? ORDER BY score DESC LIMIT ?`, mode, n)
}

// GetByName returns every entry for a player, best score first.
func (r *LeaderboardRepository) GetByName(name string) ([]LeaderboardEntry, error) {
	return r.query(selectEntries+` WHERE name = ? ORDER BY score DESC`, name)
}

func (r *LeaderboardRepository) Delete(id int) error {
	const query = `DELETE FROM leaderboard WHERE id = ?`
	res, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete leaderboard entry: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm deletion: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("no entry found with id '%d'", id)
	}

	return nil
}

func (r *LeaderboardRepository) query(query string, args ...any) ([]LeaderboardEntry, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Score, &e.Level, &e.Mode, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leaderboard row iteration error: %w", err)
	}

	return entries, nil
}
