package ledger

import "fmt"

// PostTransaction validates and records a transaction for an account.
func PostTransaction(store Store, accountID string, amount int64, description string) (Transaction, error) {
	if amount == 0 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrAmountZero)
	}

	return Transaction{}, nil
}
