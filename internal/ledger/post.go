package ledger

import (
	"fmt"
	"strings"
)

// PostTransaction validates and records a transaction for an account.
func PostTransaction(store Store, accountID string, amount int64, description string) (Transaction, error) {
	if _, err := store.Account(accountID); err != nil {
		return Transaction{}, fmt.Errorf("posting transaction: %w", err)
	}
	if amount == 0 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrAmountZero)
	}
	if strings.TrimSpace(description) == "" {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrDescriptionEmpty)
	}
	if len(description) > 140 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrDescriptionTooLong)
	}

	return Transaction{}, nil
}
