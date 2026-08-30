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

func TestRenderCSV(t *testing.T) {
	firstCreatedAt := time.Date(2026, time.August, 30, 14, 5, 6, 0, time.FixedZone("UTC-4", -4*60*60))
	secondCreatedAt := time.Date(2026, time.August, 30, 18, 5, 7, 0, time.UTC)

	tests := []struct {
		name         string
		transactions []ledger.Transaction
		want         string
		wantRecords  [][]string
	}{
		{
			name: "transactions preserve input order and render signed cents with UTC timestamps",
			transactions: []ledger.Transaction{
				{ID: "txn-2", Amount: -125, Description: "Rent", CreatedAt: firstCreatedAt},
				{ID: "txn-1", Amount: 2500, Description: "Paycheck", CreatedAt: secondCreatedAt},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-2,-125,Rent,2026-08-30T18:05:06Z\n" +
				"txn-1,2500,Paycheck,2026-08-30T18:05:07Z\n",
		},
		{
			name: "CSV descriptions containing commas quotes and newlines round trip as four fields",
			transactions: []ledger.Transaction{
				{ID: "txn-special", Amount: -1, Description: "Dinner, \"late\"\nwith friends", CreatedAt: secondCreatedAt},
			},
			want: "id,amount_cents,description,created_at\n" +
				"txn-special,-1,\"Dinner, \"\"late\"\"\nwith friends\",2026-08-30T18:05:07Z\n",
			wantRecords: [][]string{
				{"id", "amount_cents", "description", "created_at"},
				{"txn-special", "-1", "Dinner, \"late\"\nwith friends", "2026-08-30T18:05:07Z"},
			},
		},
		{
			name: "empty input renders the header only",
			want: "id,amount_cents,description,created_at\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderCSV(tt.transactions)
			if gotString := string(got); gotString != tt.want {
				t.Fatalf("string(renderCSV()) = %q, want %q", gotString, tt.want)
			}

			if tt.wantRecords == nil {
				return
			}
			records, err := csv.NewReader(strings.NewReader(string(got))).ReadAll()
			if err != nil {
				t.Fatalf("renderCSV() output is not CSV: %v", err)
			}
			if !reflect.DeepEqual(records, tt.wantRecords) {
				t.Errorf("renderCSV() records = %#v, want %#v", records, tt.wantRecords)
			}
		})
	}

	transactions := []ledger.Transaction{{ID: "txn-1", Amount: 1, Description: "same", CreatedAt: secondCreatedAt}}
	if first, second := renderCSV(transactions), renderCSV(transactions); !bytes.Equal(first, second) {
		t.Errorf("repeated renderCSV() output differs: first %q, second %q", string(first), string(second))
	}
}
