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

func TestRenderTransactionsCSVProducesDeterministicFourFieldDocument(t *testing.T) {
	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         csvRenderResult
	}{
		{
			name: "populated input preserves values order and special descriptions",
			transactions: []ledger.Transaction{
				{ID: "txn-newest", Amount: 158579, Description: "comma, value", CreatedAt: time.Date(2026, 8, 13, 14, 5, 6, 0, time.FixedZone("EDT", -4*60*60))},
				{ID: "txn-middle", Amount: -6450, Description: "a \"quoted\" value", CreatedAt: time.Date(2026, 8, 13, 17, 4, 5, 0, time.UTC)},
				{ID: "txn-oldest", Amount: 1, Description: "line one\nline two", CreatedAt: time.Date(2026, 8, 13, 18, 3, 4, 0, time.FixedZone("CEST", 2*60*60))},
			},
			want: csvRenderResult{
				document: "id,amount_cents,description,created_at\n" +
					"txn-newest,158579,\"comma, value\",2026-08-13T18:05:06Z\n" +
					"txn-middle,-6450,\"a \"\"quoted\"\" value\",2026-08-13T17:04:05Z\n" +
					"txn-oldest,1,\"line one\nline two\",2026-08-13T16:03:04Z\n",
				records: [][]string{
					{"id", "amount_cents", "description", "created_at"},
					{"txn-newest", "158579", "comma, value", "2026-08-13T18:05:06Z"},
					{"txn-middle", "-6450", "a \"quoted\" value", "2026-08-13T17:04:05Z"},
					{"txn-oldest", "1", "line one\nline two", "2026-08-13T16:03:04Z"},
				},
				fieldCounts: []int{4, 4, 4, 4},
				repeated:    true,
			},
		},
		{
			name:         "empty input produces header only",
			transactions: nil,
			want: csvRenderResult{
				document:    "id,amount_cents,description,created_at\n",
				records:     [][]string{{"id", "amount_cents", "description", "created_at"}},
				fieldCounts: []int{4},
				repeated:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, firstErr := renderTransactionsCSV(tt.transactions)
			second, secondErr := renderTransactionsCSV(tt.transactions)
			records, parseErr := csv.NewReader(strings.NewReader(string(first))).ReadAll()
			fieldCounts := make([]int, len(records))
			for i := range records {
				fieldCounts[i] = len(records[i])
			}
			got := csvRenderResult{
				document:    string(first),
				records:     records,
				fieldCounts: fieldCounts,
				repeated:    bytes.Equal(first, second),
				firstErr:    firstErr,
				secondErr:   secondErr,
				parseErr:    parseErr,
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderTransactionsCSV() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

type csvRenderResult struct {
	document    string
	records     [][]string
	fieldCounts []int
	repeated    bool
	firstErr    error
	secondErr   error
	parseErr    error
}
