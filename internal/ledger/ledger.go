package ledger

import "time"

type Account struct {
	ID   string
	Name string
}

type Transaction struct {
	ID          string
	AccountID   string
	Amount      int64
	Description string
	CreatedAt   time.Time
}
