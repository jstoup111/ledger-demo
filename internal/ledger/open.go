package ledger

import "fmt"

// OpenAccount creates an account with the next sequential account ID.
func OpenAccount(store Store, name string) (Account, error) {
	accounts, err := store.Accounts()
	if err != nil {
		return Account{}, err
	}

	account := Account{
		ID:   fmt.Sprintf("acct-%d", len(accounts)+1),
		Name: name,
	}
	if err := store.InsertAccount(account); err != nil {
		return Account{}, err
	}
	return account, nil
}
