// Package data persists local leaderboard entries in SQLite.
package data

import (
	"database/sql"
	"fmt"

	// Import the sqlite driver.
	_ "modernc.org/sqlite"
)

// NewDB opens (creating if needed) the SQLite database at dataSourceName and
// applies the schema.
func NewDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}

	if err := EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// EnsureSchema creates the tables and indexes if they don't exist yet.
func EnsureSchema(db *sql.DB) error {
	const query = `
		CREATE TABLE IF NOT EXISTS leaderboard (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     NOT NULL,
			score      INTEGER  NOT NULL DEFAULT 0,
			level      INTEGER  NOT NULL DEFAULT 1,
			mode       TEXT     NOT NULL DEFAULT 'normal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_leaderboard_score ON leaderboard (score DESC);
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("creating schema: %w", err)
	}
	return nil
}
