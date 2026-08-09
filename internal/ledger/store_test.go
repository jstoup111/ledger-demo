package ledger

type fakeStore struct{}

func (fakeStore) Accounts() ([]Account, error) {
	return nil, nil
}

func (fakeStore) Account(string) (Account, error) {
	return Account{}, nil
}

func (fakeStore) Transactions(string) ([]Transaction, error) {
	return nil, nil
}

func (fakeStore) CountTransactions() (int, error) {
	return 0, nil
}

func (fakeStore) Append(Transaction) error {
	return nil
}

var _ Store = (*fakeStore)(nil)
