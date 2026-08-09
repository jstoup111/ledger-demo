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
	// daysBeforeSeed is a fixed, deterministic offset (in whole days) counting
	// backwards from the seed instant. It gives each row its own "when recorded"
	// value instead of every seeded row sharing one timestamp. One pair on
	// acct-1 deliberately shares an offset (see demoSeed) so the store's
	// `ORDER BY created_at DESC, id DESC` tiebreak stays exercised by real seed
	// data rather than becoming dead code.
	daysBeforeSeed int
}

// loadSeedData loads the fixed, plausible data shown in the live demo. Every
// transaction's recorded time is derived from baseClock — never from the
// wall clock — by offsetting the seed instant per row, so two `make reset`
// runs stay byte-identical.
func loadSeedData(baseClock clock.Clock, database *store.SQLite) error {
	seedInstant := baseClock.Now()
	for _, seed := range demoSeed {
		if err := database.InsertAccount(seed.account); err != nil {
			return fmt.Errorf("insert seed account %q: %w", seed.account.ID, err)
		}
		for _, transaction := range seed.transactions {
			recordedAt := seedInstant.AddDate(0, 0, -transaction.daysBeforeSeed)
			transactionClock := clock.FixedClock{T: recordedAt}
			if _, err := ledger.PostTransaction(transactionClock, database, seed.account.ID, transaction.amount, transaction.description); err != nil {
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
			{amount: 158579, description: "Paycheck", daysBeforeSeed: 11},
			{amount: -6450, description: "Grocery market", daysBeforeSeed: 10},
			{amount: -3200, description: "Electric bill", daysBeforeSeed: 9},
			{amount: -1499, description: "Music subscription", daysBeforeSeed: 8},
			{amount: -7850, description: "Neighborhood cafe", daysBeforeSeed: 7},
			{amount: 2500, description: "Cashback reward", daysBeforeSeed: 6},
			{amount: -4250, description: "Fuel station", daysBeforeSeed: 5},
			{amount: -1880, description: "Phone bill", daysBeforeSeed: 4},
			{amount: -7600, description: "Dinner with friends", daysBeforeSeed: 3},
			{amount: 10000, description: "Tax refund", daysBeforeSeed: 2},
			{amount: -4500, description: "Membership renewal", daysBeforeSeed: 1},
			// Deliberately shares its offset with the row above: acct-1's newest
			// two transactions tie on created_at, so the created_at DESC, id DESC
			// tiebreak has real seed-data coverage (see TestLoadSeedDataTiebreak...
			// in seed_test.go) instead of only being tested with synthetic rows.
			{amount: -5500, description: "Concert tickets", daysBeforeSeed: 1},
		},
	},
	{
		account: ledger.Account{ID: "acct-2", Name: "Weekend Savings"},
		transactions: []seedTransaction{
			{amount: 850000, description: "Opening balance", daysBeforeSeed: 8},
			{amount: 30000, description: "Monthly savings contribution", daysBeforeSeed: 7},
			{amount: 30000, description: "Travel envelope deposit", daysBeforeSeed: 6},
			{amount: 30000, description: "Weekend plans savings", daysBeforeSeed: 5},
			{amount: -12000, description: "Train tickets", daysBeforeSeed: 4},
			{amount: 30000, description: "Cabin weekend savings", daysBeforeSeed: 3},
			{amount: -5400, description: "Museum passes", daysBeforeSeed: 2},
			{amount: 30000, description: "Next adventure savings", daysBeforeSeed: 1},
			{amount: -22500, description: "Cabin deposit", daysBeforeSeed: 0},
		},
	},
	{
		account: ledger.Account{ID: "acct-3", Name: "Project Fund"},
	},
}
