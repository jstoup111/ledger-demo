package httpapi

import (
	"errors"
	"math"
	"testing"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int64
		wantErr error
	}{
		{name: "whole dollars", input: "25", want: 2500},
		{name: "dollars and cents", input: "3.50", want: 350},
		{name: "negative dollars and cents", input: "-42.50", want: -4250},
		{name: "one cent", input: "0.01", want: 1},
		{name: "zero", input: "0", want: 0},
		{name: "largest positive cent amount", input: "92233720368547758.07", want: math.MaxInt64},
		{name: "largest negative cent amount", input: "-92233720368547758.08", want: math.MinInt64},
		{name: "letters are rejected", input: "abc", wantErr: ledger.ErrAmountMalformed},
		{name: "multiple decimal points are rejected", input: "1.2.3", wantErr: ledger.ErrAmountMalformed},
		{name: "commas are rejected", input: "1,000", wantErr: ledger.ErrAmountMalformed},
		{name: "currency symbols are rejected", input: "$5", wantErr: ledger.ErrAmountMalformed},
		{name: "more than two decimal places are rejected", input: "1.234", wantErr: ledger.ErrAmountMalformed},
		{name: "empty string is rejected", input: "", wantErr: ledger.ErrAmountMalformed},
		{name: "whitespace is rejected", input: "  ", wantErr: ledger.ErrAmountMalformed},
		{name: "leading plus is rejected", input: "+5", wantErr: ledger.ErrAmountMalformed},
		{name: "signed fractional component is rejected", input: "1.+5", wantErr: ledger.ErrAmountMalformed},
		{name: "signed fractional component after dollars is rejected", input: "25.+5", wantErr: ledger.ErrAmountMalformed},
		{name: "positive cents one beyond int64 is rejected", input: "92233720368547758.08", wantErr: ledger.ErrAmountMalformed},
		{name: "negative cents one beyond int64 is rejected", input: "-92233720368547758.09", wantErr: ledger.ErrAmountMalformed},
		{name: "positive whole dollars beyond int64 is rejected", input: "92233720368547759", wantErr: ledger.ErrAmountMalformed},
		{name: "negative whole dollars beyond int64 is rejected", input: "-92233720368547759", wantErr: ledger.ErrAmountMalformed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmount(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseAmount(%q) error = %v, want errors.Is(..., %v)", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseAmount(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
