package ledger

import (
	"errors"
	"strings"
	"testing"
)

type postingStore struct {
	fakeStore
	appended     []Transaction
	accountErr   error
	transactions []Transaction
}

func (s *postingStore) Account(id string) (Account, error) {
	if s.accountErr != nil {
		return Account{}, s.accountErr
	}
	return s.fakeStore.Account(id)
}

func (s *postingStore) Append(transaction Transaction) error {
	s.appended = append(s.appended, transaction)
	return nil
}

func (s *postingStore) Transactions(string) ([]Transaction, error) {
	return s.transactions, nil
}

func TestPostTransactionRejectsUnknownAccountBeforeOtherValidation(t *testing.T) {
	store := &postingStore{accountErr: ErrAccountNotFound}

	_, err := PostTransaction(store, "missing-account", 0, "   ")

	if !errors.Is(err, ErrAccountNotFound) || len(store.appended) != 0 {
		t.Fatalf("PostTransaction() error = %v, appended = %d; want ErrAccountNotFound and no transaction", err, len(store.appended))
	}
}

func TestPostTransactionEnforcesBalanceFloor(t *testing.T) {
	tests := []struct {
		name        string
		amount      int64
		wantErr     error
		wantBalance int64
	}{
		{name: "would make balance negative", amount: -1001, wantErr: ErrBalanceWouldGoNegative, wantBalance: 1000},
		{name: "brings balance to zero", amount: -1000, wantBalance: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &postingStore{transactions: []Transaction{{Amount: 1000}}}
			balance, err := Balance(store, "acct-1")
			if err != nil {
				t.Fatalf("Balance() error = %v", err)
			}

			_, err = PostTransaction(store, "acct-1", tt.amount, "withdrawal")

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("PostTransaction() error = %v, want error matching %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && len(store.appended) != 0 {
				t.Fatalf("PostTransaction() appended = %d; want no transaction", len(store.appended))
			}
			if err == nil {
				balance += tt.amount
			}
			if balance != tt.wantBalance {
				t.Fatalf("recomputed balance = %d, want %d", balance, tt.wantBalance)
			}
		})
	}
}

func TestPostTransactionRejectsZeroAmount(t *testing.T) {
	store := &postingStore{}

	_, err := PostTransaction(store, "acct-1", 0, "deposit")

	if !errors.Is(err, ErrAmountZero) || len(store.appended) != 0 {
		t.Fatalf("PostTransaction() error = %v, appended = %d; want ErrAmountZero and no transaction", err, len(store.appended))
	}
}

func TestPostTransactionRejectsEmptyDescription(t *testing.T) {
	tests := []struct {
		name        string
		description string
	}{
		{name: "empty", description: ""},
		{name: "spaces", description: "   "},
		{name: "whitespace", description: "\t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &postingStore{}

			_, err := PostTransaction(store, "acct-1", 100, tt.description)

			if !errors.Is(err, ErrDescriptionEmpty) || len(store.appended) != 0 {
				t.Fatalf("PostTransaction() error = %v, appended = %d; want ErrDescriptionEmpty and no transaction", err, len(store.appended))
			}
		})
	}
}

func TestPostTransactionEnforcesDescriptionLengthLimit(t *testing.T) {
	tests := []struct {
		name        string
		description string
		wantErr     error
	}{
		{name: "maximum length", description: strings.Repeat("a", 140)},
		{name: "over maximum length", description: strings.Repeat("a", 141), wantErr: ErrDescriptionTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &postingStore{}

			_, err := PostTransaction(store, "acct-1", 100, tt.description)

			if !errors.Is(err, tt.wantErr) || len(store.appended) != 0 {
				t.Fatalf("PostTransaction() error = %v, appended = %d; want error matching %v and no transaction", err, len(store.appended), tt.wantErr)
			}
		})
	}
}
