package httpapi

import (
	"encoding/csv"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderTransactionsCSVProducesDeterministicDocument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		transactions []ledger.Transaction
		wantDocument string
		wantRecords  [][]string
	}{
		{
			name: "preserves ordered signed amounts timestamps and escaped descriptions",
			transactions: []ledger.Transaction{
				{
					ID:          "txn-positive",
					Amount:      1250,
					Description: "deposit, with \"memo\"\nand second line",
					CreatedAt:   time.Date(2026, time.August, 14, 11, 4, 5, 0, time.FixedZone("UTC-4", -4*60*60)),
				},
				{
					ID:          "txn-negative",
					Amount:      -275,
					Description: "withdrawal",
					CreatedAt:   time.Date(2026, time.August, 14, 16, 5, 6, 0, time.UTC),
				},
			},
			wantDocument: "id,amount_cents,description,created_at\n" +
				"txn-positive,1250,\"deposit, with \"\"memo\"\"\nand second line\",2026-08-14T15:04:05Z\n" +
				"txn-negative,-275,withdrawal,2026-08-14T16:05:06Z\n",
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-positive", "1250", "deposit, with \"memo\"\nand second line", "2026-08-14T15:04:05Z"},
				{"txn-negative", "-275", "withdrawal", "2026-08-14T16:05:06Z"},
			},
		},
		{
			name:         "emits only the header for empty input",
			transactions: []ledger.Transaction{},
			wantDocument: "id,amount_cents,description,created_at\n",
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, firstErr := renderTransactionsCSV(tt.transactions)
			second, secondErr := renderTransactionsCSV(tt.transactions)
			records, parseErr := csv.NewReader(strings.NewReader(string(first))).ReadAll()

			got := struct {
				firstDocument  string
				secondDocument string
				records        [][]string
				firstErr       error
				secondErr      error
				parseErr       error
			}{string(first), string(second), records, firstErr, secondErr, parseErr}
			want := struct {
				firstDocument  string
				secondDocument string
				records        [][]string
				firstErr       error
				secondErr      error
				parseErr       error
			}{tt.wantDocument, tt.wantDocument, tt.wantRecords, nil, nil, nil}

			if !reflect.DeepEqual(got, want) {
				t.Errorf("renderTransactionsCSV() = %#v, want %#v", got, want)
			}
		})
	}
}
