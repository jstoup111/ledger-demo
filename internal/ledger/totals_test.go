package ledger

import (
	"errors"
	"math"
	"testing"
)

func amounts(values ...int64) []Transaction {
	transactions := make([]Transaction, len(values))
	for i, value := range values {
		transactions[i] = Transaction{Amount: value}
	}
	return transactions
}

func TestTotalsOfSumsDepositsAndWithdrawals(t *testing.T) {
	tests := []struct {
		name         string
		transactions []Transaction
		want         Totals
	}{
		{name: "empty list", want: Totals{}},
		{
			name:         "ten 10-cent deposits total exactly 100",
			transactions: amounts(10, 10, 10, 10, 10, 10, 10, 10, 10, 10),
			want:         Totals{Deposits: 100, Withdrawals: 0, Net: 100},
		},
		{
			name:         "mixed deposits and withdrawals",
			transactions: amounts(10000, 2550, -4025),
			want:         Totals{Deposits: 12550, Withdrawals: -4025, Net: 8525},
		},
		{
			name:         "withdrawals only",
			transactions: amounts(-300, -200),
			want:         Totals{Deposits: 0, Withdrawals: -500, Net: -500},
		},
		{
			name:         "zero amount counts toward neither side",
			transactions: amounts(0, 5),
			want:         Totals{Deposits: 5, Withdrawals: 0, Net: 5},
		},
		{
			name:         "extremes cancel in net without overflow",
			transactions: amounts(math.MaxInt64, math.MinInt64),
			want:         Totals{Deposits: math.MaxInt64, Withdrawals: math.MinInt64, Net: -1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TotalsOf(tt.transactions)
			if err != nil {
				t.Fatalf("TotalsOf() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("TotalsOf() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTotalsOfRejectsInt64Overflow(t *testing.T) {
	tests := []struct {
		name         string
		transactions []Transaction
	}{
		{name: "deposits above max", transactions: amounts(math.MaxInt64, 1)},
		{name: "withdrawals below min", transactions: amounts(math.MinInt64, -1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TotalsOf(tt.transactions)
			if !errors.Is(err, ErrBalanceOverflow) {
				t.Fatalf("TotalsOf() error = %v, want ErrBalanceOverflow", err)
			}
			if got != (Totals{}) {
				t.Fatalf("TotalsOf() = %+v, want zero Totals on error", got)
			}
		})
	}
}
