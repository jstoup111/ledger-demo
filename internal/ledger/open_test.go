package ledger

import (
	"reflect"
	"testing"
)

type openStore struct {
	accounts     []Account
	transactions map[string][]Transaction
}

func (s *openStore) Accounts() ([]Account, error) {
	return append([]Account(nil), s.accounts...), nil
}

func (s *openStore) Account(id string) (Account, error) {
	for _, account := range s.accounts {
		if account.ID == id {
			return account, nil
		}
	}
	return Account{}, ErrAccountNotFound
}

func (s *openStore) Transactions(accountID string) ([]Transaction, error) {
	if _, err := s.Account(accountID); err != nil {
		return nil, err
	}
	return append([]Transaction(nil), s.transactions[accountID]...), nil
}

func (s *openStore) CountTransactions() (int, error) { return 0, nil }

func (s *openStore) Append(Transaction) error { return nil }

func (s *openStore) InsertAccount(account Account) error {
	s.accounts = append(s.accounts, account)
	return nil
}

func TestOpenAccount(t *testing.T) {
	seededAccounts := []Account{
		{ID: "acct-1", Name: "Checking"},
		{ID: "acct-2", Name: "Savings"},
		{ID: "acct-3", Name: "Travel"},
	}

	tests := []struct {
		name         string
		initial      []Account
		names        []string
		wantOpened   []Account
		wantAccounts []Account
	}{
		{
			name:       "opens after seeded accounts with an empty ledger",
			initial:    seededAccounts,
			names:      []string{"Holiday Fund"},
			wantOpened: []Account{{ID: "acct-4", Name: "Holiday Fund"}},
			wantAccounts: append(seededAccounts,
				Account{ID: "acct-4", Name: "Holiday Fund"},
			),
		},
		{
			name:       "allows duplicate names with sequential IDs",
			initial:    seededAccounts,
			names:      []string{"Holiday Fund", "Holiday Fund"},
			wantOpened: []Account{{ID: "acct-4", Name: "Holiday Fund"}, {ID: "acct-5", Name: "Holiday Fund"}},
			wantAccounts: append(seededAccounts,
				Account{ID: "acct-4", Name: "Holiday Fund"},
				Account{ID: "acct-5", Name: "Holiday Fund"},
			),
		},
		{
			name:         "opens the first account in an empty store",
			names:        []string{"Holiday Fund"},
			wantOpened:   []Account{{ID: "acct-1", Name: "Holiday Fund"}},
			wantAccounts: []Account{{ID: "acct-1", Name: "Holiday Fund"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &openStore{accounts: append([]Account(nil), tt.initial...)}
			var gotOpened []Account
			for _, name := range tt.names {
				account, err := OpenAccount(store, name)
				if err != nil {
					t.Fatalf("OpenAccount() error = %v", err)
				}
				gotOpened = append(gotOpened, account)
			}

			if !reflect.DeepEqual(gotOpened, tt.wantOpened) {
				t.Errorf("opened accounts = %#v, want %#v", gotOpened, tt.wantOpened)
			}
			accounts, err := store.Accounts()
			if err != nil {
				t.Fatalf("Accounts() error = %v", err)
			}
			if !reflect.DeepEqual(accounts, tt.wantAccounts) {
				t.Errorf("Accounts() = %#v, want %#v", accounts, tt.wantAccounts)
			}
			for _, account := range gotOpened {
				balance, err := Balance(store, account.ID)
				if err != nil || balance != 0 {
					t.Errorf("Balance(%q) = %d, %v; want 0, nil", account.ID, balance, err)
				}
				transactions, err := store.Transactions(account.ID)
				if err != nil || len(transactions) != 0 {
					t.Errorf("Transactions(%q) = %#v, %v; want empty, nil", account.ID, transactions, err)
				}
			}
		})
	}
}
