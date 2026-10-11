// Covers: S1.1, S1.2, S1.3, S1.4, S1.5, S2.1, S2.2, S2.3, S3.1, S3.2
package ledger

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

type openStore struct {
	accounts      []Account
	transactions  map[string][]Transaction
	accountsErr   error
	insertErr     error
	accountsCalls int
	insertCalls   int
}

func (s *openStore) Accounts() ([]Account, error) {
	s.accountsCalls++
	if s.accountsErr != nil {
		return nil, s.accountsErr
	}
	return append([]Account(nil), s.accounts...), nil
}

func TestOpenAccountRejectsBlankNames(t *testing.T) {
	seededAccounts := []Account{
		{ID: "acct-1", Name: "Checking"},
		{ID: "acct-2", Name: "Savings"},
		{ID: "acct-3", Name: "Travel"},
	}

	tests := []struct {
		name        string
		accountName string
	}{
		{name: "empty", accountName: ""},
		{name: "spaces only", accountName: "   "},
		{name: "mixed whitespace", accountName: " \t\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &openStore{accounts: append([]Account(nil), seededAccounts...)}

			account, err := OpenAccount(store, tt.accountName)

			if account != (Account{}) || !errors.Is(err, ErrAccountNameEmpty) || len(store.accounts) != 3 || store.accountsCalls != 0 || store.insertCalls != 0 {
				t.Fatalf("OpenAccount(%q) = %#v, %v; accounts = %d, Accounts calls = %d, InsertAccount calls = %d; want zero account, ErrAccountNameEmpty, three accounts, and no store calls", tt.accountName, account, err, len(store.accounts), store.accountsCalls, store.insertCalls)
			}
		})
	}
}

func TestOpenAccountPreservesSurroundingWhitespace(t *testing.T) {
	const name = "  Rainy Day  "
	store := &openStore{}

	account, err := OpenAccount(store, name)

	if err != nil || account.Name != name || len(store.accounts) != 1 || store.accounts[0].Name != name {
		t.Fatalf("OpenAccount(%q) = %#v, %v; stored accounts = %#v; want preserved name", name, account, err, store.accounts)
	}
}

func TestOpenAccountLimitsNamesToFortyCharacters(t *testing.T) {
	seededAccounts := []Account{
		{ID: "acct-1", Name: "Checking"},
		{ID: "acct-2", Name: "Savings"},
		{ID: "acct-3", Name: "Travel"},
	}

	tests := []struct {
		name        string
		accountName string
		wantErr     error
	}{
		{name: "40 ASCII characters", accountName: strings.Repeat("a", 40)},
		{name: "41 ASCII characters", accountName: strings.Repeat("a", 41), wantErr: ErrAccountNameTooLong},
		{name: "40 e acute characters", accountName: strings.Repeat("é", 40)},
		{name: "41 e acute characters", accountName: strings.Repeat("é", 41), wantErr: ErrAccountNameTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &openStore{accounts: append([]Account(nil), seededAccounts...)}

			account, err := OpenAccount(store, tt.accountName)

			if tt.wantErr != nil {
				if account != (Account{}) || !errors.Is(err, tt.wantErr) || len(store.accounts) != 3 || store.accountsCalls != 0 || store.insertCalls != 0 {
					t.Fatalf("OpenAccount(%q) = %#v, %v; accounts = %d, Accounts calls = %d, InsertAccount calls = %d; want zero account, ErrAccountNameTooLong, three accounts, and no store calls", tt.accountName, account, err, len(store.accounts), store.accountsCalls, store.insertCalls)
				}
				return
			}

			wantAccount := Account{ID: "acct-4", Name: tt.accountName}
			if err != nil || account != wantAccount || len(store.accounts) != 4 || store.insertCalls != 1 {
				t.Fatalf("OpenAccount(%q) = %#v, %v; accounts = %#v, InsertAccount calls = %d; want %#v, nil, four accounts, and one insert", tt.accountName, account, err, store.accounts, store.insertCalls, wantAccount)
			}
		})
	}
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
	s.insertCalls++
	if s.insertErr != nil {
		return s.insertErr
	}
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

func TestOpenAccountStorageFailures(t *testing.T) {
	storageErr := errors.New("storage unavailable")
	insertErr := errors.New("insert unavailable")
	accountNameEmpty := errors.New("account name empty")
	accountNameTooLong := errors.New("account name too long")

	t.Run("does not insert when listing accounts fails", func(t *testing.T) {
		store := &openStore{accountsErr: storageErr}

		account, err := OpenAccount(store, "Holiday Fund")

		if account != (Account{}) || err.Error() != "opening account: storage unavailable" || !errors.Is(err, storageErr) || errors.Is(err, accountNameEmpty) || errors.Is(err, accountNameTooLong) || store.insertCalls != 0 {
			t.Fatalf("OpenAccount() = %#v, %v; insert calls = %d; want zero account, storage error only, and zero insert calls", account, err, store.insertCalls)
		}
	})

	t.Run("returns no account when inserting fails", func(t *testing.T) {
		store := &openStore{insertErr: insertErr}

		account, err := OpenAccount(store, "Holiday Fund")

		if account != (Account{}) || err.Error() != "opening account: insert unavailable" || !errors.Is(err, insertErr) {
			t.Fatalf("OpenAccount() = %#v, %v; want zero account and insert error", account, err)
		}
	})
}
