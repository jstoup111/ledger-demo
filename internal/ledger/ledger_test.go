package ledger

import (
	"testing"
	"time"
)

func TestTransactionLiteralRetainsAllFields(t *testing.T) {
	createdAt := time.Date(2026, time.August, 8, 12, 0, 0, 0, time.UTC)
	transaction := Transaction{
		ID:          "txn-123",
		AccountID:   "account-456",
		Amount:      2500,
		Description: "initial deposit",
		CreatedAt:   createdAt,
	}
	var _ int64 = transaction.Amount

	want := Transaction{
		ID:          "txn-123",
		AccountID:   "account-456",
		Amount:      2500,
		Description: "initial deposit",
		CreatedAt:   createdAt,
	}
	if transaction != want {
		t.Fatalf("Transaction literal = %#v, want %#v", transaction, want)
	}
}
