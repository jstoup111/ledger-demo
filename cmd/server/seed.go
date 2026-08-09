package main

import (
	"fmt"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

type seedAccount struct {
	account      ledger.Account
	transactions []seedTransaction
}

type seedTransaction struct {
	amount      int64
	description string
}

// loadSeedData loads the fixed, plausible data shown in the live demo.
func loadSeedData(clock clock.Clock, database *store.SQLite) error {
	for _, seed := range demoSeed {
		if err := database.InsertAccount(seed.account); err != nil {
			return fmt.Errorf("insert seed account %q: %w", seed.account.ID, err)
		}
		for _, transaction := range seed.transactions {
			if _, err := ledger.PostTransaction(clock, database, seed.account.ID, transaction.amount, transaction.description); err != nil {
				return fmt.Errorf("post seed transaction for %q: %w", seed.account.ID, err)
			}
		}
	}
	return nil
}

var demoSeed = []seedAccount{
	{
		account: ledger.Account{ID: "acct-1", Name: "Everyday Checking"},
		transactions: []seedTransaction{
			{amount: 245000, description: "Paycheck"},
			{amount: -6450, description: "Grocery market"},
			{amount: -3200, description: "Electric bill"},
			{amount: -1499, description: "Music subscription"},
			{amount: -7850, description: "Neighborhood cafe"},
			{amount: 2500, description: "Cashback reward"},
			{amount: -4250, description: "Fuel station"},
			{amount: -1880, description: "Phone bill"},
			{amount: -7600, description: "Dinner with friends"},
		},
	},
	{
		account: ledger.Account{ID: "acct-2", Name: "Weekend Savings"},
		transactions: []seedTransaction{
			{amount: 850000, description: "Opening balance"},
			{amount: 30000, description: "Automatic transfer"},
			{amount: 30000, description: "Automatic transfer"},
			{amount: 30000, description: "Automatic transfer"},
			{amount: -12000, description: "Train tickets"},
			{amount: 30000, description: "Automatic transfer"},
			{amount: -5400, description: "Museum passes"},
			{amount: 30000, description: "Automatic transfer"},
			{amount: -22500, description: "Cabin deposit"},
		},
	},
	{
		account: ledger.Account{ID: "acct-3", Name: "Project Fund"},
		transactions: []seedTransaction{
			{amount: 180000, description: "Project budget"},
			{amount: -24000, description: "Materials shop"},
			{amount: -6800, description: "Printing"},
			{amount: 12500, description: "Client reimbursement"},
			{amount: -19500, description: "Workshop rental"},
			{amount: -3600, description: "Courier"},
			{amount: 48000, description: "Milestone payment"},
			{amount: -7900, description: "Reference books"},
			{amount: -11200, description: "Presentation supplies"},
		},
	},
}
