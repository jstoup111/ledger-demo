package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
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
	idPattern := regexp.MustCompile(`^txn-\d{4}$`)
	seenIDs := make(map[string]bool, len(first.transactions))
	perAccount := make(map[string][]ledger.Transaction, len(first.accounts))
	anchor := seedClock.Now()
	hasAnchor := false
	earliest := first.transactions[0].CreatedAt
	latest := first.transactions[0].CreatedAt
	for _, transaction := range first.transactions {
		if !idPattern.MatchString(transaction.ID) {
			t.Fatalf("transaction ID %q does not match %q", transaction.ID, idPattern)
		}
		if seenIDs[transaction.ID] {
			t.Fatalf("transaction ID %q is duplicated", transaction.ID)
		}
		seenIDs[transaction.ID] = true
		perAccount[transaction.AccountID] = append(perAccount[transaction.AccountID], transaction)
		if transaction.CreatedAt.After(anchor) {
			t.Fatalf("transaction %q created at %v, after anchor %v", transaction.ID, transaction.CreatedAt, anchor)
		}
		if transaction.CreatedAt.Equal(anchor) {
			hasAnchor = true
		}
		if !transaction.CreatedAt.Truncate(time.Second).Equal(transaction.CreatedAt) {
			t.Fatalf("transaction %q created at %v, want whole-second precision", transaction.ID, transaction.CreatedAt)
		}
		if !strings.HasSuffix(transaction.CreatedAt.Format(time.RFC3339Nano), "Z") {
			t.Fatalf("transaction %q created at %v, want UTC/Z", transaction.ID, transaction.CreatedAt)
		}
		if transaction.CreatedAt.Before(earliest) {
			earliest = transaction.CreatedAt
		}
		if transaction.CreatedAt.After(latest) {
			latest = transaction.CreatedAt
		}
	}
	if !hasAnchor {
		t.Fatal("no seeded transaction has the anchor created-at time")
	}
	if got := latest.Sub(earliest); got < 28*24*time.Hour {
		t.Fatalf("seeded timestamp span = %v, want at least 28 days", got)
	}
	for accountID, transactions := range perAccount {
		if len(transactions) == 0 {
			continue
		}
		sort.Slice(transactions, func(i, j int) bool {
			return transactions[i].ID < transactions[j].ID
		})
		for i := 1; i < len(transactions); i++ {
			if transactions[i].CreatedAt.Before(transactions[i-1].CreatedAt) {
				t.Fatalf("%s transaction %q created at %v, before preceding %q at %v", accountID, transactions[i].ID, transactions[i].CreatedAt, transactions[i-1].ID, transactions[i-1].CreatedAt)
			}
		}
	}
	acct1Times := make(map[time.Time]bool, len(perAccount["acct-1"]))
	for _, transaction := range perAccount["acct-1"] {
		acct1Times[transaction.CreatedAt] = true
	}
	if len(acct1Times) < 2 {
		t.Fatalf("acct-1 distinct created-at values = %d, want more than 1", len(acct1Times))
	}
	acct2Times := make(map[time.Time]bool, len(perAccount["acct-2"]))
	for _, transaction := range perAccount["acct-2"] {
		acct2Times[transaction.CreatedAt] = true
	}
	if got, want := len(acct2Times), len(perAccount["acct-2"]); got != want {
		t.Fatalf("acct-2 distinct created-at values = %d, want %d", got, want)
	}
	var acct1Balance int64
	for _, transaction := range perAccount["acct-1"] {
		acct1Balance += transaction.Amount
	}
	if acct1Balance != 128350 {
		t.Fatalf("acct-1 balance = %d cents, want 128350", acct1Balance)
	}
	for _, transaction := range perAccount["acct-2"] {
		for _, forbidden := range []string{"transfer", "interest", "fee"} {
			if regexp.MustCompile(`(?i)` + forbidden).MatchString(transaction.Description) {
				t.Fatalf("acct-2 transaction %q uses forbidden non-goal term %q", transaction.Description, forbidden)
			}
		}
	}
	if len(first.transactions) < 16 || len(first.transactions) > 24 {
		t.Fatalf("seed transactions = %d, want 16-24", len(first.transactions))
	}
	// FR-15, amended 2026-08-09: the first two accounts carry 8-12 transactions
	// and acct-3 is seeded EMPTY, so FR-4's empty-history state is reachable on
	// stage. Nothing in this system can create an account, and draining a balance
	// to zero leaves a populated list, so an empty seeded account is the only way
	// that state exists. This guard is deliberately per-account rather than a bare
	// total: a total-only check is what let a fixture violating the per-account
	// shape pass green in earlier cycles.
	for _, account := range first.accounts {
		got := len(perAccount[account.ID])
		if account.ID == "acct-3" {
			if got != 0 {
				t.Fatalf("account %q transactions = %d, want 0 — it is seeded empty", account.ID, got)
			}
			continue
		}
		if got < 8 || got > 12 {
			t.Fatalf("account %q transactions = %d, want 8-12", account.ID, got)
		}
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
