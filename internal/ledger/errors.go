package ledger

import "errors"

var (
	ErrAccountNotFound        = errors.New("account not found")
	ErrAmountZero             = errors.New("amount must not be zero")
	ErrDescriptionEmpty       = errors.New("description must not be empty")
	ErrDescriptionTooLong     = errors.New("description is too long")
	ErrAmountMalformed        = errors.New("amount is malformed")
	ErrBalanceWouldGoNegative = errors.New("balance would go negative")
)
