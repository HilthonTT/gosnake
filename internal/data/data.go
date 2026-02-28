package data

import (
	"database/sql"

	// Import the sqlite driver.
	_ "modernc.org/sqlite"
)

func NewDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}

	if err := EnsureTableExists(db); err != nil {
		return nil, err
	}

	return db, nil
}

func EnsureTableExists(db *sql.DB) error {
	const query = `
		CREATE TABLE IF NOT EXISTS leaderboard (
			id         INTEGER  PRIMARY KEY AUTOINCREMENT,
			name       TEXT     NOT NULL,
			score      INTEGER  NOT NULL DEFAULT 0,
			level      INTEGER  NOT NULL DEFAULT 1,
			mode       TEXT     NOT NULL DEFAULT 'normal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(query)
	return err
}
