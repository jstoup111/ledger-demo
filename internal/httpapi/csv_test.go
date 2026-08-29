// Covers: task:1
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
	createdAt := time.Date(2026, time.August, 28, 14, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         csvRender
	}{
		{
			name: "renders transactions in input order with signed cent amounts and UTC timestamps",
			transactions: []ledger.Transaction{
				{ID: "9", Amount: 1250, Description: "Deposit", CreatedAt: createdAt},
				{ID: "3", Amount: -499, Description: "Groceries", CreatedAt: createdAt.Add(time.Minute)},
			},
			want: csvRender{
				bytes: "id,amount_cents,description,created_at\n9,1250,Deposit,2026-08-28T18:30:00Z\n3,-499,Groceries,2026-08-28T18:31:00Z\n",
				records: [][]string{
					{"id", "amount_cents", "description", "created_at"},
					{"9", "1250", "Deposit", "2026-08-28T18:30:00Z"},
					{"3", "-499", "Groceries", "2026-08-28T18:31:00Z"},
				},
				deterministic: true,
			},
		},
		{
			name: "quotes descriptions that contain CSV special characters without changing them",
			transactions: []ledger.Transaction{
				{ID: "1", Amount: 0, Description: "Coffee, \"tea\"\nwater", CreatedAt: time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)},
			},
			want: csvRender{
				bytes: "id,amount_cents,description,created_at\n1,0,\"Coffee, \"\"tea\"\"\nwater\",2026-01-02T03:04:05Z\n",
				records: [][]string{
					{"id", "amount_cents", "description", "created_at"},
					{"1", "0", "Coffee, \"tea\"\nwater", "2026-01-02T03:04:05Z"},
				},
				deterministic: true,
			},
		},
		{
			name: "renders an empty transaction list as only the header",
			want: csvRender{
				bytes:         "id,amount_cents,description,created_at\n",
				records:       [][]string{{"id", "amount_cents", "description", "created_at"}},
				deterministic: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := observeCSV(t, renderTransactionsCSV(tt.transactions), renderTransactionsCSV(tt.transactions))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderTransactionsCSV() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

type csvRender struct {
	bytes         string
	records       [][]string
	deterministic bool
}

func observeCSV(t *testing.T, first, second []byte) csvRender {
	t.Helper()
	records, err := csv.NewReader(strings.NewReader(string(first))).ReadAll()
	if err != nil {
		t.Fatalf("parse rendered CSV: %v", err)
	}
	return csvRender{bytes: string(first), records: records, deterministic: bytes.Equal(first, second)}
}
