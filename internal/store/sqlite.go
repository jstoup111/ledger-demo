package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

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

// Close releases the database connection pool.
func (s *SQLite) Close() error {
	return s.db.Close()
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

// Append stores a transaction.
func (s *SQLite) Append(transaction ledger.Transaction) error {
	createdAt := transaction.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00")
	if _, err := s.db.Exec(`
		INSERT INTO transactions (id, account_id, amount, description, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, transaction.ID, transaction.AccountID, transaction.Amount, transaction.Description, createdAt); err != nil {
		return fmt.Errorf("append transaction %q: %w", transaction.ID, err)
	}
	return nil
}

// Transactions returns an account's transactions newest first.
func (s *SQLite) Transactions(accountID string) ([]ledger.Transaction, error) {
	if _, err := s.Account(accountID); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(`
		SELECT id, account_id, amount, description, created_at
		FROM transactions
		WHERE account_id = ?
		ORDER BY created_at DESC, id DESC
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("list transactions for account %q: %w", accountID, err)
	}
	defer rows.Close()

	transactions := make([]ledger.Transaction, 0)
	for rows.Next() {
		var transaction ledger.Transaction
		var createdAt string
		if err := rows.Scan(
			&transaction.ID,
			&transaction.AccountID,
			&transaction.Amount,
			&transaction.Description,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		transaction.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse transaction %q created at: %w", transaction.ID, err)
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return transactions, nil
}

// CountTransactions returns the total number of transactions across all accounts.
func (s *SQLite) CountTransactions() (int, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count transactions: %w", err)
	}
	return count, nil
}
