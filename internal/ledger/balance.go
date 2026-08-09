package ledger

// Balance derives an account's balance by summing its transactions.
func Balance(store Store, accountID string) (int64, error) {
	transactions, err := store.Transactions(accountID)
	if err != nil {
		return 0, err
	}

	var balance int64
	for _, transaction := range transactions {
		balance += transaction.Amount
	}

	return balance, nil
}
