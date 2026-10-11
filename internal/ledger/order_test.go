// Covers: task:11
package ledger

import (
	"reflect"
	"testing"
)

func TestSortAccountsOrdersIDsByLengthThenText(t *testing.T) {
	tests := []struct {
		name     string
		accounts []Account
		want     []Account
	}{
		{
			name: "orders double digit IDs after single digit IDs",
			accounts: []Account{
				{ID: "acct-10"},
				{ID: "acct-2"},
				{ID: "acct-12"},
				{ID: "acct-1"},
				{ID: "acct-9"},
			},
			want: []Account{
				{ID: "acct-1"},
				{ID: "acct-2"},
				{ID: "acct-9"},
				{ID: "acct-10"},
				{ID: "acct-12"},
			},
		},
		{
			name: "orders equal length IDs by text",
			accounts: []Account{
				{ID: "acct-3"},
				{ID: "acct-1"},
				{ID: "acct-2"},
			},
			want: []Account{
				{ID: "acct-1"},
				{ID: "acct-2"},
				{ID: "acct-3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortAccounts(tt.accounts)

			if !reflect.DeepEqual(tt.accounts, tt.want) {
				t.Fatalf("SortAccounts() = %#v, want %#v", tt.accounts, tt.want)
			}
		})
	}
}
