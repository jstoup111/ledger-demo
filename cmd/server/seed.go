package main

import (
	"fmt"
	"time"

	"github.com/jstoup111/ledger-demo/internal/clock"
	"github.com/jstoup111/ledger-demo/internal/ledger"
	"github.com/jstoup111/ledger-demo/internal/store"
)

const (
	day  = 24 * time.Hour
	hour = time.Hour
)

type seedAccount struct {
	account      ledger.Account
	transactions []seedTransaction
}

type seedTransaction struct {
	amount int64
	// recordedBefore offsets are non-increasing down each account's list and never negative.
	recordedBefore time.Duration
	description    string
}

// loadSeedData loads the fixed, plausible data shown in the live demo.
func loadSeedData(seedClock clock.Clock, database *store.SQLite) error {
	anchor := seedClock.Now()
	for _, seed := range demoSeed {
		if err := database.InsertAccount(seed.account); err != nil {
			return fmt.Errorf("insert seed account %q: %w", seed.account.ID, err)
		}
		for _, transaction := range seed.transactions {
			if _, err := ledger.PostTransaction(clock.FixedClock{T: anchor.Add(-transaction.recordedBefore)}, database, seed.account.ID, transaction.amount, transaction.description); err != nil {
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
			{amount: 158579, recordedBefore: 32 * day, description: "Paycheck"},
			{amount: -6450, recordedBefore: 30*day + 3*hour, description: "Grocery market"},
			{amount: -3200, recordedBefore: 27*day + 5*hour, description: "Electric bill"},
			{amount: -1499, recordedBefore: 24*day + hour, description: "Music subscription"},
			{amount: -7850, recordedBefore: 20*day + 6*hour, description: "Neighborhood cafe"},
			{amount: 2500, recordedBefore: 17*day + 2*hour, description: "Cashback reward"},
			{amount: -4250, recordedBefore: 13*day + 7*hour, description: "Fuel station"},
			{amount: -1880, recordedBefore: 10*day + 4*hour, description: "Phone bill"},
			{amount: -7600, recordedBefore: 6*day + 5*hour, description: "Dinner with friends"},
			{amount: 10000, recordedBefore: 3*day + 3*hour, description: "Tax refund"},
			{amount: -4500, recordedBefore: day + 2*hour, description: "Membership renewal"},
			{amount: -5500, recordedBefore: 0, description: "Concert tickets"},
		},
	},
	{
		account: ledger.Account{ID: "acct-2", Name: "Weekend Savings"},
		transactions: []seedTransaction{
			{amount: 850000, recordedBefore: 60 * day, description: "Opening balance"},
			{amount: 30000, recordedBefore: 52*day + 2*hour, description: "Monthly savings contribution"},
			{amount: 30000, recordedBefore: 45*day + 4*hour, description: "Travel envelope deposit"},
			{amount: 30000, recordedBefore: 38*day + hour, description: "Weekend plans savings"},
			{amount: -12000, recordedBefore: 31*day + 6*hour, description: "Train tickets"},
			{amount: 30000, recordedBefore: 24*day + 3*hour, description: "Cabin weekend savings"},
			{amount: -5400, recordedBefore: 17*day + 5*hour, description: "Museum passes"},
			{amount: 30000, recordedBefore: 10*day + 2*hour, description: "Next adventure savings"},
			{amount: -22500, recordedBefore: 2*day + 8*hour, description: "Cabin deposit"},
		},
	},
	{
		account: ledger.Account{ID: "acct-3", Name: "Project Fund"},
	},
}
