// Covers: task:1, S2.1, S2.2, S3.1, S4.1
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
			name: "preserves values order and special descriptions",
			transactions: []ledger.Transaction{
				{
					ID:          "txn-0002",
					Amount:      1250,
					Description: "Coffee, \"morning\"\nwith a friend",
					CreatedAt:   time.Date(2026, time.August, 28, 9, 10, 11, 0, time.FixedZone("EDT", -4*60*60)),
				},
				{
					ID:          "txn-0001",
					Amount:      -425,
					Description: "Refund",
					CreatedAt:   time.Date(2026, time.August, 27, 18, 0, 0, 0, time.UTC),
				},
			},
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-0002", "1250", "Coffee, \"morning\"\nwith a friend", "2026-08-28T13:10:11Z"},
				{"txn-0001", "-425", "Refund", "2026-08-27T18:00:00Z"},
			},
		},
		{
			name:         "empty input emits only the header",
			transactions: nil,
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
			},
			wantCSV: "id,amount_cents,description,created_at\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstRender := renderTransactionsCSV(tt.transactions)
			secondRender := renderTransactionsCSV(tt.transactions)
			if string(firstRender) != string(secondRender) {
				t.Fatalf("renderTransactionsCSV() rendered unchanged input differently: first %q, second %q", firstRender, secondRender)
			}
			if tt.wantCSV != "" && string(firstRender) != tt.wantCSV {
				t.Fatalf("renderTransactionsCSV() CSV = %q, want %q", firstRender, tt.wantCSV)
			}

			reader := csv.NewReader(strings.NewReader(string(firstRender)))
			reader.FieldsPerRecord = 4
			gotRecords, err := reader.ReadAll()
			if err != nil {
				t.Fatalf("renderTransactionsCSV() produced invalid four-field CSV: %v", err)
			}
			if !reflect.DeepEqual(gotRecords, tt.wantRecords) {
				t.Fatalf("renderTransactionsCSV() records = %#v, want %#v", gotRecords, tt.wantRecords)
			}
		})
	}
}
