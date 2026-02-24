package data

import (
	"database/sql"
	"fmt"
)

type LeaderboardEntry struct {
	ID        int
	Name      string
	Score     int
	Level     int
	CreatedAt string
}

type LeaderboardRepository struct {
	db *sql.DB
}

func NewLeaderboardRepository(db *sql.DB) *LeaderboardRepository {
	return &LeaderboardRepository{db}
}

func (r *LeaderboardRepository) Save(name string, score, level int) (int, error) {
	const query = `
		INSERT INTO leaderboard (name, score, level)
		VALUES (?, ?, ?)
	`
	res, err := r.db.Exec(query, name, score, level)
	if err != nil {
		return 0, fmt.Errorf("failed to save leaderboard entry: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to save leaderboard entry: %w", err)
	}

	return int(id), nil
}

func (r *LeaderboardRepository) All() ([]LeaderboardEntry, error) {
	const query = `
		SELECT id, name, score, level, created_at
		FROM leaderboard
		ORDER BY score DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Score, &e.Level, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leaderboard row iteration error: %w", err)
	}

	return entries, nil
}

func (r *LeaderboardRepository) GetTopN(n int) ([]LeaderboardEntry, error) {
	const query = `
		SELECT id, name, score, level, created_at
		FROM leaderboard
		ORDER BY score DESC
		LIMIT ?
	`
	rows, err := r.db.Query(query, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Score, &e.Level, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leaderboard row iteration error: %w", err)
	}

	return entries, nil
}

func (r *LeaderboardRepository) GetByName(name string) ([]LeaderboardEntry, error) {
	const query = `
		SELECT id, name, score, level, created_at
		FROM leaderboard
		WHERE name = ?
		ORDER BY score DESC
	`
	rows, err := r.db.Query(query, name)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard by name: %w", err)
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		if err := rows.Scan(&e.ID, &e.Name, &e.Score, &e.Level, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("leaderboard row iteration error: %w", err)
	}

	return entries, nil
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
