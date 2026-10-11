package ledger

import "sort"

// SortAccounts orders accounts by ID length, then ID text.
func SortAccounts(accounts []Account) {
	sort.SliceStable(accounts, func(i, j int) bool {
		if len(accounts[i].ID) != len(accounts[j].ID) {
			return len(accounts[i].ID) < len(accounts[j].ID)
		}
		return accounts[i].ID < accounts[j].ID
	})
}
