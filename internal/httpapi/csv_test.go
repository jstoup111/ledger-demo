// Covers: task:1
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
	firstRecorded := time.Date(2026, time.August, 8, 14, 31, 0, 0, time.FixedZone("EDT", -4*60*60))
	secondRecorded := time.Date(2026, time.August, 8, 14, 30, 0, 0, time.UTC)
	transactions := []ledger.Transaction{
		{
			ID:          "txn-0002",
			AccountID:   "acct-1",
			Amount:      -4250,
			Description: "Groceries, \"fresh\"\nweekly",
			CreatedAt:   firstRecorded,
		},
		{
			ID:          "txn-0001",
			AccountID:   "acct-1",
			Amount:      128350,
			Description: "Deposit",
			CreatedAt:   secondRecorded,
		},
	}

	for _, tt := range []struct {
		name         string
		transactions []ledger.Transaction
		wantRecords  [][]string
	}{
		{
			name:         "preserves header signed cents UTC timestamps order and escaped descriptions",
			transactions: transactions,
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-0002", "-4250", "Groceries, \"fresh\"\nweekly", "2026-08-08T18:31:00Z"},
				{"txn-0001", "128350", "Deposit", "2026-08-08T14:30:00Z"},
			},
		},
		{
			name:        "empty input emits only the header",
			wantRecords: [][]string{{"id", "amount_cents", "description", "created_at"}},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTransactionsCSV(tt.transactions)
			records, err := csv.NewReader(strings.NewReader(string(got))).ReadAll()
			if err != nil {
				t.Fatalf("rendered CSV does not parse: %v; body = %q", err, got)
			}
			for index, record := range records {
				if got, want := len(record), 4; got != want {
					t.Errorf("record %d fields = %d, want %d; record = %#v", index, got, want, record)
				}
			}
			if !reflect.DeepEqual(records, tt.wantRecords) {
				t.Errorf("CSV records = %#v, want %#v", records, tt.wantRecords)
			}
		})
	}

	first := renderTransactionsCSV(transactions)
	second := renderTransactionsCSV(transactions)
	if !reflect.DeepEqual(first, second) {
		t.Errorf("repeated renders differ: first = %q, second = %q", first, second)
	}
}
