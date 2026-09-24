package httpapi

import (
	"testing"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestAverageDollars(t *testing.T) {
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         float64
		wantOK       bool
	}{
		{
			name:         "1000 and 2000 cents average to 15",
			transactions: []ledger.Transaction{{Amount: 1000}, {Amount: 2000}},
			want:         15,
			wantOK:       true,
		},
		{name: "empty list has no average", transactions: nil, wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := averageDollars(test.transactions)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if got != test.want {
				t.Fatalf("average = %v, want %v", got, test.want)
			}
		})
	}
}

func TestFormatAverage(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{name: "whole dollars get two decimals", input: 15, want: "15.00"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatAverage(test.input); got != test.want {
				t.Fatalf("formatAverage(%v) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
