package httpapi

import (
	"bytes"
	"encoding/csv"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

// Covers: task:1 S2/S3/S4.
func TestRenderTransactionsCSV(t *testing.T) {
	utc := time.Date(2026, time.August, 28, 14, 5, 6, 0, time.UTC)
	transactions := []ledger.Transaction{
		{ID: "txn-credit", Amount: 1250, Description: "paycheck, August", CreatedAt: utc},
		{ID: "txn-debit", Amount: -375, Description: "say \"hello\"\nthen leave", CreatedAt: utc.In(time.FixedZone("EDT", -4*60*60))},
		{ID: "txn-max", Amount: math.MaxInt64, Description: "largest credit", CreatedAt: utc},
		{ID: "txn-min", Amount: math.MinInt64, Description: "largest debit", CreatedAt: utc},
	}

	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         [][]string
	}{
		{
			name:         "signed cents, UTC times, and escaped descriptions preserve input order",
			transactions: transactions,
			want: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-credit", "1250", "paycheck, August", "2026-08-28T14:05:06Z"},
				{"txn-debit", "-375", "say \"hello\"\nthen leave", "2026-08-28T14:05:06Z"},
				{"txn-max", "9223372036854775807", "largest credit", "2026-08-28T14:05:06Z"},
				{"txn-min", "-9223372036854775808", "largest debit", "2026-08-28T14:05:06Z"},
			},
		},
		{
			name: "empty input produces only the header",
			want: [][]string{
				{"id", "amount_cents", "description", "created_at"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderTransactionsCSV(tt.transactions)
			reader := csv.NewReader(bytes.NewReader(got))
			records, err := reader.ReadAll()
			if err != nil {
				t.Fatalf("renderTransactionsCSV() emitted invalid CSV: %v", err)
			}
			if !reflect.DeepEqual(records, tt.want) {
				t.Errorf("renderTransactionsCSV() records = %#v, want %#v", records, tt.want)
			}
			for index, record := range records {
				if len(record) != 4 {
					t.Errorf("record %d has %d fields, want 4", index, len(record))
				}
			}
		})
	}

	first := renderTransactionsCSV(transactions)
	second := renderTransactionsCSV(transactions)
	if !bytes.Equal(first, second) {
		t.Errorf("renderTransactionsCSV() produced different bytes on repeated calls:\nfirst:  %q\nsecond: %q", first, second)
	}
}
