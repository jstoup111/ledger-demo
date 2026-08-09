package store

import (
	"database/sql"
	"fmt"

	// The pure-Go SQLite driver (no CGO), registered with database/sql under
	// the name "sqlite".
	//
	// This blank import is here before any code uses it on purpose: it pins
	// modernc.org/sqlite in go.mod and go.sum so the module is already resolved
	// and cached. The demo must build and run with no network, and discovering a
	// missing dependency on stage is not recoverable.
	_ "modernc.org/sqlite"
)

// SQLite stores ledger data in a SQLite database.
type SQLite struct {
	db *sql.DB
}

// Open opens a SQLite database at dsn and creates the ledger schema.
func Open(dsn string) (*SQLite, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	if dsn == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL REFERENCES accounts(id),
			amount INTEGER NOT NULL,
			description TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create ledger schema: %w", err)
	}

	return &SQLite{db: db}, nil
}
