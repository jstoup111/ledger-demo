package main

import (
	"fmt"
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
	if len(first.transactions) < 24 || len(first.transactions) > 36 {
		t.Fatalf("seed transactions = %d, want 24-36", len(first.transactions))
	}

	idPattern := regexp.MustCompile(`^txn-\d{4}$`)
	seenIDs := make(map[string]bool, len(first.transactions))
	perAccount := make(map[string]int, len(first.accounts))
	for _, transaction := range first.transactions {
		if !idPattern.MatchString(transaction.ID) {
			t.Fatalf("transaction ID %q does not match %q", transaction.ID, idPattern)
		}
		if seenIDs[transaction.ID] {
			t.Fatalf("transaction ID %q is duplicated", transaction.ID)
		}
		seenIDs[transaction.ID] = true
		perAccount[transaction.AccountID]++
		if transaction.CreatedAt != seedClock.Now() {
			t.Fatalf("transaction %q created at %v, want injected clock %v", transaction.ID, transaction.CreatedAt, seedClock.Now())
		}
	}
	for _, account := range first.accounts {
		if count := perAccount[account.ID]; count < 8 || count > 12 {
			t.Fatalf("account %q has %d transactions, want 8-12", account.ID, count)
		}
	}
	for number := 1; number <= len(first.transactions); number++ {
		id := fmt.Sprintf("txn-%04d", number)
		if !seenIDs[id] {
			t.Fatalf("transaction IDs are not globally unbroken: missing %q", id)
		}
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
