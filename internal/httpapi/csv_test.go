package httpapi

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderTransactionsCSV(t *testing.T) {
	newest := time.Date(2026, time.August, 9, 17, 30, 45, 0, time.FixedZone("EDT", -4*60*60))
	older := time.Date(2026, time.August, 8, 12, 5, 6, 0, time.FixedZone("PDT", -7*60*60))

	tests := []struct {
		name         string
		transactions []ledger.Transaction
		wantRecords  [][]string
	}{
		{
			name: "values retain input order and use CSV-safe fields",
			transactions: []ledger.Transaction{
				{ID: "newest", Amount: 158579, Description: "comma, quote \" and\nline break", CreatedAt: newest},
				{ID: "older", Amount: -6450, Description: "withdrawal", CreatedAt: older},
			},
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"newest", "158579", "comma, quote \" and\nline break", "2026-08-09T21:30:45Z"},
				{"older", "-6450", "withdrawal", "2026-08-08T19:05:06Z"},
			},
		},
		{
			name:         "empty input produces header only",
			transactions: nil,
			wantRecords:  [][]string{{"id", "amount_cents", "description", "created_at"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, err := renderTransactionsCSV(tt.transactions)
			if err != nil {
				t.Fatalf("renderTransactionsCSV() error = %v", err)
			}
			second, err := renderTransactionsCSV(tt.transactions)
			if err != nil {
				t.Fatalf("second renderTransactionsCSV() error = %v", err)
			}
			if !bytes.Equal(first, second) {
				t.Fatalf("two renders differ:\nfirst:  %q\nsecond: %q", first, second)
			}

			records, err := csv.NewReader(strings.NewReader(string(first))).ReadAll()
			if err != nil {
				t.Fatalf("parse rendered CSV: %v", err)
			}
			if !reflect.DeepEqual(records, tt.wantRecords) {
				t.Errorf("rendered records = %#v, want %#v", records, tt.wantRecords)
			}
		})
	}
}
