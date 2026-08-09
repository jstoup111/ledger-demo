package ledger

import (
	"errors"
	"math"
	"testing"
)

type balanceStore struct {
	fakeStore
	transactions []Transaction
}

func (s balanceStore) Transactions(string) ([]Transaction, error) {
	return s.transactions, nil
}

func TestBalanceSumsTransactions(t *testing.T) {
	tests := []struct {
		name         string
		transactions []Transaction
		want         int64
	}{
		{name: "empty log", want: 0},
		{
			name: "single deposit",
			transactions: []Transaction{
				{Amount: 2500},
			},
			want: 2500,
		},
		{
			name: "deposit and withdrawal",
			transactions: []Transaction{
				{Amount: 128350},
				{Amount: -4250},
			},
			want: 124100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Balance(balanceStore{transactions: tt.transactions}, "acct-1")
			if err != nil {
				t.Fatalf("Balance() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Balance() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBalanceRejectsInt64Overflow(t *testing.T) {
	_, err := Balance(balanceStore{transactions: []Transaction{{Amount: math.MaxInt64}, {Amount: 1}}}, "acct-1")
	if !errors.Is(err, ErrBalanceOverflow) {
		t.Fatalf("Balance() error = %v, want ErrBalanceOverflow", err)
	}
}
