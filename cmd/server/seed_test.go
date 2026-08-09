package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

var seedClock = clock.FixedClock{T: time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)}

func TestLoadSeedDataIsDeterministic(t *testing.T) {
	first := seedSnapshot(t)
	second := seedSnapshot(t)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("two fresh seed snapshots differ:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first.accounts) != 3 {
		t.Fatalf("seed accounts = %d, want 3", len(first.accounts))
	}
	if got, want := []string{first.accounts[0].ID, first.accounts[1].ID, first.accounts[2].ID}, []string{"acct-1", "acct-2", "acct-3"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("seed account IDs = %v, want %v", got, want)
	}
	if len(first.transactions) < 24 || len(first.transactions) > 36 {
		t.Fatalf("seed transactions = %d, want 24-36", len(first.transactions))
	}

	idPattern := regexp.MustCompile(`^txn-\d{4}$`)
	seenIDs := make(map[string]bool, len(first.transactions))
	perAccount := make(map[string][]ledger.Transaction, len(first.accounts))
	for _, transaction := range first.transactions {
		if !idPattern.MatchString(transaction.ID) {
			t.Fatalf("transaction ID %q does not match %q", transaction.ID, idPattern)
		}
		if seenIDs[transaction.ID] {
			t.Fatalf("transaction ID %q is duplicated", transaction.ID)
		}
		seenIDs[transaction.ID] = true
		perAccount[transaction.AccountID] = append(perAccount[transaction.AccountID], transaction)
		if transaction.CreatedAt != seedClock.Now() {
			t.Fatalf("transaction %q created at %v, want injected clock %v", transaction.ID, transaction.CreatedAt, seedClock.Now())
		}
	}
	var acct1Balance int64
	for _, transaction := range perAccount["acct-1"] {
		acct1Balance += transaction.Amount
	}
	if acct1Balance != 128350 {
		t.Fatalf("acct-1 balance = %d cents, want 128350", acct1Balance)
	}
	if got := len(perAccount["acct-3"]); got != 0 {
		t.Fatalf("acct-3 transactions = %d, want 0", got)
	}
	for number := 1; number <= len(first.transactions); number++ {
		id := fmt.Sprintf("txn-%04d", number)
		if !seenIDs[id] {
			t.Fatalf("transaction IDs are not globally unbroken: missing %q", id)
		}
	}
}

func TestSeedResetProducesIdenticalFileBackedRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ledger.db")
	if !filepath.IsAbs(dbPath) {
		t.Fatalf("database path %q is not absolute", dbPath)
	}
	t.Setenv("LEDGER_DB_PATH", dbPath)

	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	isolatedDirectory := t.TempDir()
	if err := os.Chdir(isolatedDirectory); err != nil {
		t.Fatalf("Chdir(%q) error = %v", isolatedDirectory, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Errorf("restore working directory %q: %v", workingDirectory, err)
		}
	})
	defaultPath := filepath.Join(isolatedDirectory, "ledger.db")

	if err := run("seed"); err != nil {
		t.Fatalf("first run(seed) error = %v", err)
	}
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("default database %q was touched; stat error = %v", defaultPath, err)
	}
	first := seededDatabaseState(t, dbPath)

	if err := os.Remove(dbPath); err != nil {
		t.Fatalf("Remove(%q) error = %v", dbPath, err)
	}

	if err := run("seed"); err != nil {
		t.Fatalf("second run(seed) error = %v", err)
	}
	if _, err := os.Stat(defaultPath); !os.IsNotExist(err) {
		t.Fatalf("default database %q was touched; stat error = %v", defaultPath, err)
	}
	second := seededDatabaseState(t, dbPath)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("seed rows differ after file-backed reset:\nfirst:  %#v\nsecond: %#v", first, second)
	}
}

type seedState struct {
	accounts     []ledger.Account
	transactions []ledger.Transaction
}

func seedSnapshot(t *testing.T) seedState {
	t.Helper()
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}

	if err := loadSeedData(seedClock, database); err != nil {
		t.Fatalf("loadSeedData() error = %v", err)
	}

	accounts, err := database.Accounts()
	if err != nil {
		t.Fatalf("Accounts() error = %v", err)
	}
	transactions := make([]ledger.Transaction, 0)
	for _, account := range accounts {
		rows, err := database.Transactions(account.ID)
		if err != nil {
			t.Fatalf("Transactions(%q) error = %v", account.ID, err)
		}
		transactions = append(transactions, rows...)
	}
	return seedState{accounts: accounts, transactions: transactions}
}
