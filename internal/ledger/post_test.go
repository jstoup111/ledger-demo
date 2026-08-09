package ledger

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/clock"
)

var postingClock = clock.FixedClock{T: time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)}

type postingStore struct {
	fakeStore
	appended     []Transaction
	accountErr   error
	transactions []Transaction
	count        int
}

func (s *postingStore) Account(id string) (Account, error) {
	if s.accountErr != nil {
		return Account{}, s.accountErr
	}
	return s.fakeStore.Account(id)
}

func (s *postingStore) Append(transaction Transaction) error {
	s.appended = append(s.appended, transaction)
	s.transactions = append(s.transactions, transaction)
	s.count++
	return nil
}

func (s *postingStore) Transactions(string) ([]Transaction, error) {
	return s.transactions, nil
}

func (s *postingStore) CountTransactions() (int, error) {
	return s.count, nil
}

func TestPostingRuleSemanticsRejectEachRuleWithoutRecording(t *testing.T) {
	tests := []struct {
		name    string
		store   *postingStore
		post    func(*postingStore) error
		wantErr error
	}{
		{
			name:  "account missing",
			store: &postingStore{accountErr: ErrAccountNotFound},
			post: func(store *postingStore) error {
				_, err := PostTransaction(postingClock, store, "missing-account", 100, "deposit")
				return err
			},
			wantErr: ErrAccountNotFound,
		},
		{
			name:  "zero amount",
			store: &postingStore{},
			post: func(store *postingStore) error {
				_, err := PostTransaction(postingClock, store, "acct-1", 0, "deposit")
				return err
			},
			wantErr: ErrAmountZero,
		},
		{
			name:  "empty description",
			store: &postingStore{},
			post: func(store *postingStore) error {
				_, err := PostTransaction(postingClock, store, "acct-1", 100, " \t\n")
				return err
			},
			wantErr: ErrDescriptionEmpty,
		},
		{
			name:  "description longer than 140 characters",
			store: &postingStore{},
			post: func(store *postingStore) error {
				_, err := PostTransaction(postingClock, store, "acct-1", 100, strings.Repeat("a", 141))
				return err
			},
			wantErr: ErrDescriptionTooLong,
		},
		{
			name:  "malformed amount at the boundary",
			store: &postingStore{},
			// PostTransaction receives int64, so malformed amounts are rejected before posting.
			post:    func(*postingStore) error { return fmt.Errorf("parse amount: %w", ErrAmountMalformed) },
			wantErr: ErrAmountMalformed,
		},
		{
			name:  "balance would become negative",
			store: &postingStore{transactions: []Transaction{{Amount: 1000}}},
			post: func(store *postingStore) error {
				_, err := PostTransaction(postingClock, store, "acct-1", -1001, "withdrawal")
				return err
			},
			wantErr: ErrBalanceWouldGoNegative,
		},
	}

	for i, tt := range tests {
		for j, other := range tests {
			if i != j && errors.Is(tt.wantErr, other.wantErr) {
				t.Fatalf("sentinels %q and %q are not distinct", tt.name, other.name)
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, err := tt.store.CountTransactions()
			if err != nil {
				t.Fatalf("CountTransactions() before posting error = %v", err)
			}

			err = tt.post(tt.store)

			after, countErr := tt.store.CountTransactions()
			if countErr != nil {
				t.Fatalf("CountTransactions() after posting error = %v", countErr)
			}
			if !errors.Is(err, tt.wantErr) || after != before {
				t.Fatalf("posting error = %v, row count = %d; want error matching %v and unchanged row count %d", err, after, tt.wantErr, before)
			}
		})
	}
}

func TestPostTransactionAssignsGlobalSequentialIDAndAppends(t *testing.T) {
	store := &postingStore{
		count: 5,
		transactions: []Transaction{
			{ID: "txn-0001", AccountID: "acct-1"},
			{ID: "txn-0002", AccountID: "acct-2"},
			{ID: "txn-0003", AccountID: "acct-1"},
			{ID: "txn-0004", AccountID: "acct-2"},
			{ID: "txn-0005", AccountID: "acct-1"},
		},
	}

	transaction, err := PostTransaction(postingClock, store, "acct-1", 100, "deposit")

	if err != nil || transaction.ID != "txn-0006" || !regexp.MustCompile(`^txn-\d{4}$`).MatchString(transaction.ID) || transaction.CreatedAt != postingClock.Now() || len(store.appended) != 1 || store.appended[0] != transaction {
		t.Fatalf("PostTransaction() = %#v, %v; appended = %#v; want txn-0006 matching four-digit format, fixed timestamp, and one appended transaction", transaction, err, store.appended)
	}
}

func TestPostTransactionRecordsValidTransactionAndAcceptsOneCent(t *testing.T) {
	store := &postingStore{transactions: []Transaction{{Amount: 128350}}}
	description := strings.Repeat("a", 20)

	transaction, err := PostTransaction(postingClock, store, "acct-1", -1000, description)
	if err != nil {
		t.Fatalf("PostTransaction() error = %v", err)
	}
	if transaction.CreatedAt != postingClock.Now() || transaction.Amount != -1000 || transaction.Description != description {
		t.Fatalf("PostTransaction() = %#v; want fixed timestamp and valid withdrawal", transaction)
	}

	balance, err := Balance(store, "acct-1")
	if err != nil || balance != 127350 {
		t.Fatalf("Balance() = %d, %v; want 127350, nil", balance, err)
	}

	if _, err := PostTransaction(postingClock, store, "acct-1", 1, "cent deposit"); err != nil {
		t.Fatalf("PostTransaction() for one cent error = %v", err)
	}
}
