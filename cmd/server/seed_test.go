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
		// Every seeded timestamp must be derived from the injected seed clock,
		// never wall time: it can't be later than the seed instant, and the gap
		// back to the seed instant must land on a whole-day boundary produced by
		// a fixed per-row offset rather than an arbitrary wall-clock value.
		delta := seedClock.Now().Sub(transaction.CreatedAt)
		if delta < 0 {
			t.Fatalf("transaction %q created at %v is after the injected seed instant %v", transaction.ID, transaction.CreatedAt, seedClock.Now())
		}
		if delta%(24*time.Hour) != 0 {
			t.Fatalf("transaction %q created at %v is not a whole-day offset from the injected seed instant %v; timestamps must be derived from the clock, not wall time", transaction.ID, transaction.CreatedAt, seedClock.Now())
		}
		if delta > 30*24*time.Hour {
			t.Fatalf("transaction %q created at %v is implausibly far (%v) from the injected seed instant %v", transaction.ID, transaction.CreatedAt, delta, seedClock.Now())
		}
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

// TestLoadSeedDataTimestampsAreDistinctExceptOneDeliberateTie guards the fix for
// seeded rows all sharing one timestamp: recorded times must differ across a
// funded account's transactions, with exactly one intentional exception (on
// acct-1) kept so the created_at DESC, id DESC tiebreak in
// internal/store.SQLite.Transactions stays covered by real seed data instead of
// becoming dead code.
func TestLoadSeedDataTimestampsAreDistinctExceptOneDeliberateTie(t *testing.T) {
	snapshot := seedSnapshot(t)

	perAccount := make(map[string][]ledger.Transaction)
	for _, transaction := range snapshot.transactions {
		perAccount[transaction.AccountID] = append(perAccount[transaction.AccountID], transaction)
	}

	for accountID, transactions := range perAccount {
		byTimestamp := make(map[time.Time][]string)
		for _, transaction := range transactions {
			byTimestamp[transaction.CreatedAt] = append(byTimestamp[transaction.CreatedAt], transaction.ID)
		}

		tiedGroups := 0
		for _, ids := range byTimestamp {
			if len(ids) > 1 {
				tiedGroups++
			}
		}

		switch accountID {
		case "acct-1":
			if tiedGroups != 1 {
				t.Fatalf("account %q has %d groups of tied timestamps, want exactly 1 deliberate tie", accountID, tiedGroups)
			}
		default:
			if tiedGroups != 0 {
				t.Fatalf("account %q has %d groups of tied timestamps, want 0 (all recorded times distinct)", accountID, tiedGroups)
			}
		}
	}
}

// TestLoadSeedDataTiebreakOrdersByIDDescendingOnEqualTimestamps exercises the
// real seed fixture end to end through the store's newest-first query, so the
// created_at DESC, id DESC tiebreak has coverage from actual seed data (acct-1's
// deliberately tied pair) and not only from synthetic fixtures.
func TestLoadSeedDataTiebreakOrdersByIDDescendingOnEqualTimestamps(t *testing.T) {
	database, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:) error = %v", err)
	}
	if err := loadSeedData(seedClock, database); err != nil {
		t.Fatalf("loadSeedData() error = %v", err)
	}

	transactions, err := database.Transactions("acct-1")
	if err != nil {
		t.Fatalf("Transactions(%q) error = %v", "acct-1", err)
	}
	if len(transactions) < 2 {
		t.Fatalf("acct-1 transactions = %d, want at least 2", len(transactions))
	}

	newest, secondNewest := transactions[0], transactions[1]
	if newest.CreatedAt != secondNewest.CreatedAt {
		t.Fatalf("expected the two most recent acct-1 transactions to share a timestamp; got %v (%s) and %v (%s)", newest.CreatedAt, newest.ID, secondNewest.CreatedAt, secondNewest.ID)
	}
	if newest.ID != "txn-0012" || secondNewest.ID != "txn-0011" {
		t.Fatalf("tied-timestamp rows ordered as %s, %s; want txn-0012 before txn-0011 per id DESC tiebreak", newest.ID, secondNewest.ID)
	}
	if len(transactions) > 2 && transactions[2].CreatedAt == newest.CreatedAt {
		t.Fatalf("expected only one tied pair at the top of acct-1's history; transaction %s also shares the newest timestamp", transactions[2].ID)
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
