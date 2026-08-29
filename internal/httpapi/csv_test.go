// Covers: task:1
package httpapi

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"testing"
	"time"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

func TestRenderTransactionsCSVEncodesDeterministicParseableTransactionRows(t *testing.T) {
	transactions := []ledger.Transaction{
		{
			ID:          "txn-credit",
			Amount:      123456,
			Description: "Paycheck, \"August\"\nnet",
			CreatedAt:   time.Date(2026, time.August, 8, 10, 30, 0, 0, time.FixedZone("EDT", -4*60*60)),
		},
		{
			ID:          "txn-debit",
			Amount:      -999,
			Description: "Coffee",
			CreatedAt:   time.Date(2026, time.August, 8, 14, 31, 2, 0, time.UTC),
		},
	}

	type outcome struct {
		first               [][]string
		second              [][]string
		empty               [][]string
		isByteDeterministic bool
	}
	first := renderTransactionsCSV(transactions)
	second := renderTransactionsCSV(transactions)
	got := outcome{
		first:               readCSV(t, first),
		second:              readCSV(t, second),
		empty:               readCSV(t, renderTransactionsCSV(nil)),
		isByteDeterministic: bytes.Equal(first, second),
	}
	want := outcome{
		first: [][]string{
			{"id", "amount_cents", "description", "created_at"},
			{"txn-credit", "123456", "Paycheck, \"August\"\nnet", "2026-08-08T14:30:00Z"},
			{"txn-debit", "-999", "Coffee", "2026-08-08T14:31:02Z"},
		},
		second: [][]string{
			{"id", "amount_cents", "description", "created_at"},
			{"txn-credit", "123456", "Paycheck, \"August\"\nnet", "2026-08-08T14:30:00Z"},
			{"txn-debit", "-999", "Coffee", "2026-08-08T14:31:02Z"},
		},
		empty:               [][]string{{"id", "amount_cents", "description", "created_at"}},
		isByteDeterministic: true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("renderTransactionsCSV() parsed rows = %#v, want %#v", got, want)
	}
}

func readCSV(t *testing.T, input []byte) [][]string {
	t.Helper()
	rows, err := csv.NewReader(bytes.NewReader(input)).ReadAll()
	if err != nil {
		t.Fatalf("CSV output does not parse: %v", err)
	}
	return rows
}
