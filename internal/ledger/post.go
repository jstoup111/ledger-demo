package ledger

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/jstoup111/ledger-demo/internal/clock"
)

// PostTransaction validates and records a transaction for an account.
func PostTransaction(clock clock.Clock, store Store, accountID string, amount int64, description string) (Transaction, error) {
	if _, err := store.Account(accountID); err != nil {
		return Transaction{}, fmt.Errorf("posting transaction: %w", err)
	}
	if amount == 0 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrAmountZero)
	}
	if strings.TrimSpace(description) == "" {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrDescriptionEmpty)
	}
	if utf8.RuneCountInString(description) > 140 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrDescriptionTooLong)
	}
	balance, err := Balance(store, accountID)
	if err != nil {
		return Transaction{}, fmt.Errorf("posting transaction: %w", err)
	}
	if balance+amount < 0 {
		return Transaction{}, fmt.Errorf("posting transaction: %w", ErrBalanceWouldGoNegative)
	}

	count, err := store.CountTransactions()
	if err != nil {
		return Transaction{}, fmt.Errorf("posting transaction: %w", err)
	}
	transaction := Transaction{
		ID:          fmt.Sprintf("txn-%04d", count+1),
		AccountID:   accountID,
		Amount:      amount,
		Description: description,
		CreatedAt:   clock.Now(),
	}
	if err := store.Append(transaction); err != nil {
		return Transaction{}, fmt.Errorf("posting transaction: %w", err)
	}

	return transaction, nil
}
