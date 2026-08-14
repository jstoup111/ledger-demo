package httpapi

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderTransactionsCSV(t *testing.T) {
	newest := time.Date(2026, time.August, 14, 15, 4, 5, 0, time.FixedZone("EDT", -4*60*60))
	older := time.Date(2026, time.August, 14, 18, 3, 2, 0, time.FixedZone("CEST", 2*60*60))

	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         string
		wantRecords  [][]string
	}{
		{
			name: "newest first positive and negative cents in UTC",
			transactions: []ledger.Transaction{
				{ID: "txn-new", Amount: 12345, Description: "deposit", CreatedAt: newest},
				{ID: "txn-old", Amount: -500, Description: "withdrawal", CreatedAt: older},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-new,12345,deposit,2026-08-14T19:04:05Z\n" +
				"txn-old,-500,withdrawal,2026-08-14T16:03:02Z\n",
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-new", "12345", "deposit", "2026-08-14T19:04:05Z"},
				{"txn-old", "-500", "withdrawal", "2026-08-14T16:03:02Z"},
			},
		},
		{
			name:         "empty input has header only",
			transactions: nil,
			want:         "id,amount_cents,description,created_at\n",
			wantRecords:  [][]string{{"id", "amount_cents", "description", "created_at"}},
		},
		{
			name: "comma quote and line break descriptions remain four fields",
			transactions: []ledger.Transaction{
				{ID: "txn-special", Amount: 1, Description: "comma, quote \" and line\nbreak", CreatedAt: time.Date(2026, time.August, 14, 1, 2, 3, 0, time.UTC)},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-special,1,\"comma, quote \"\" and line\nbreak\",2026-08-14T01:02:03Z\n",
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-special", "1", "comma, quote \" and line\nbreak", "2026-08-14T01:02:03Z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTransactionsCSV(tt.transactions)
			if err != nil {
				t.Fatalf("renderTransactionsCSV() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("renderTransactionsCSV() = %q, want %q", got, tt.want)
			}

			records, err := csv.NewReader(bytes.NewReader(got)).ReadAll()
			if err != nil {
				t.Fatalf("CSV parse error = %v", err)
			}
			if !reflect.DeepEqual(records, tt.wantRecords) {
				t.Errorf("CSV records = %#v, want %#v", records, tt.wantRecords)
			}
			for _, record := range records {
				if len(record) != 4 {
					t.Errorf("CSV record has %d fields, want 4: %#v", len(record), record)
				}
			}
		})
	}
}

func TestRenderTransactionsCSVIsByteIdentical(t *testing.T) {
	transactions := []ledger.Transaction{{
		ID:          "txn-1",
		Amount:      -99,
		Description: "same bytes",
		CreatedAt:   time.Date(2026, time.August, 14, 1, 2, 3, 0, time.UTC),
	}}

	first, err := renderTransactionsCSV(transactions)
	if err != nil {
		t.Fatalf("first renderTransactionsCSV() error = %v", err)
	}
	second, err := renderTransactionsCSV(transactions)
	if err != nil {
		t.Fatalf("second renderTransactionsCSV() error = %v", err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("two renderTransactionsCSV() calls differ: %q != %q", first, second)
	}
}
