package ledger

import (
	"fmt"
	"strings"
)

// OpenAccount creates an account with the next sequential account ID.
func OpenAccount(store Store, name string) (Account, error) {
	if strings.TrimSpace(name) == "" {
		return Account{}, fmt.Errorf("opening account: %w", ErrAccountNameEmpty)
	}

	accounts, err := store.Accounts()
	if err != nil {
		return Account{}, fmt.Errorf("opening account: %w", err)
	}

	account := Account{
		ID:   fmt.Sprintf("acct-%d", len(accounts)+1),
		Name: name,
	}
	if err := store.InsertAccount(account); err != nil {
		return Account{}, fmt.Errorf("opening account: %w", err)
	}
	return account, nil
}
