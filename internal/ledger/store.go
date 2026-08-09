package ledger

// Store provides storage access for ledger accounts and transactions.
type Store interface {
	Accounts() ([]Account, error)
	Account(id string) (Account, error)
	Transactions(accountID string) ([]Transaction, error)
	CountTransactions() (int, error)
	Append(Transaction) error
}
