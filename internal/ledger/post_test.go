package ledger

import (
	"errors"
	"strings"
	"testing"
)

type postingStore struct {
	fakeStore
	appended   []Transaction
	accountErr error
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

func TestPostTransactionRejectsUnknownAccountBeforeOtherValidation(t *testing.T) {
	store := &postingStore{accountErr: ErrAccountNotFound}

	_, err := PostTransaction(store, "missing-account", 0, "   ")

	if !errors.Is(err, ErrAccountNotFound) || len(store.appended) != 0 {
		t.Fatalf("PostTransaction() error = %v, appended = %d; want ErrAccountNotFound and no transaction", err, len(store.appended))
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
