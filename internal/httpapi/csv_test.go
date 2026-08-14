package httpapi

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderTransactionsCSV(t *testing.T) {
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		wantRecords  [][]string
		wantCSV      string
	}{
		{
			name: "renders transactions in input order with CSV-safe descriptions",
			transactions: []ledger.Transaction{
				{
					ID:          "txn-credit",
					AccountID:   "account-ignored-by-export",
					Amount:      2500,
					Description: "Paycheck",
					CreatedAt:   time.Date(2026, time.January, 2, 3, 4, 5, 0, time.FixedZone("EST", -5*60*60)),
				},
				{
					ID:          "txn-debit",
					AccountID:   "account-ignored-by-export",
					Amount:      -375,
					Description: "Coffee, \"morning\"\nwith foam",
					CreatedAt:   time.Date(2026, time.January, 2, 8, 9, 10, 0, time.UTC),
				},
			},
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-credit", "2500", "Paycheck", "2026-01-02T08:04:05Z"},
				{"txn-debit", "-375", "Coffee, \"morning\"\nwith foam", "2026-01-02T08:09:10Z"},
			},
		},
		{
			name:        "renders an empty export as header only",
			wantRecords: [][]string{{"id", "amount_cents", "description", "created_at"}},
			wantCSV:     "id,amount_cents,description,created_at\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTransactionsCSV(tt.transactions)
			if repeat := renderTransactionsCSV(tt.transactions); !reflect.DeepEqual(got, repeat) {
				t.Fatalf("renderTransactionsCSV(%v) produced unstable output: first %q, second %q", tt.transactions, got, repeat)
			}
			if tt.wantCSV != "" && string(got) != tt.wantCSV {
				t.Fatalf("renderTransactionsCSV(%v) = %q, want %q", tt.transactions, got, tt.wantCSV)
			}

			records, err := csv.NewReader(strings.NewReader(string(got))).ReadAll()
			if err != nil {
				t.Fatalf("renderTransactionsCSV(%v) returned invalid CSV: %v", tt.transactions, err)
			}
			if !reflect.DeepEqual(records, tt.wantRecords) {
				t.Fatalf("renderTransactionsCSV(%v) records = %#v, want %#v", tt.transactions, records, tt.wantRecords)
			}
		})
	}
}
