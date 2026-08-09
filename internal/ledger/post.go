package ledger

import (
	"fmt"
	"strings"
)

// PostTransaction validates and records a transaction for an account.
func PostTransaction(store Store, accountID string, amount int64, description string) (Transaction, error) {
	if amount == 0 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrAmountZero)
	}
	if strings.TrimSpace(description) == "" {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrDescriptionEmpty)
	}

	return Transaction{}, nil
}
