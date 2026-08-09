package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jstoup111/ledger-demo/internal/ledger"

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

// InsertAccount stores an account.
func (s *SQLite) InsertAccount(account ledger.Account) error {
	if _, err := s.db.Exec(`INSERT INTO accounts (id, name) VALUES (?, ?)`, account.ID, account.Name); err != nil {
		return fmt.Errorf("insert account %q: %w", account.ID, err)
	}
	return nil
}

// Accounts returns all accounts ordered by ID.
func (s *SQLite) Accounts() ([]ledger.Account, error) {
	rows, err := s.db.Query(`SELECT id, name FROM accounts ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []ledger.Account
	for rows.Next() {
		var account ledger.Account
		if err := rows.Scan(&account.ID, &account.Name); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}
	return accounts, nil
}

// Account returns the account identified by id.
func (s *SQLite) Account(id string) (ledger.Account, error) {
	var account ledger.Account
	err := s.db.QueryRow(`SELECT id, name FROM accounts WHERE id = ?`, id).Scan(&account.ID, &account.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return ledger.Account{}, fmt.Errorf("account %q: %w", id, ledger.ErrAccountNotFound)
	}
	if err != nil {
		return ledger.Account{}, fmt.Errorf("read account %q: %w", id, err)
	}
	return account, nil
}
