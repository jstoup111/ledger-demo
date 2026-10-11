package ledger

import "errors"

var (
	ErrAccountNotFound        = errors.New("account not found")
	ErrAccountNameEmpty       = errors.New("account name must not be empty")
	ErrAccountNameTooLong     = errors.New("account name is too long")
	ErrAmountZero             = errors.New("amount must not be zero")
	ErrDescriptionEmpty       = errors.New("description must not be empty")
	ErrDescriptionTooLong     = errors.New("description is too long")
	ErrAmountMalformed        = errors.New("amount is malformed")
	ErrBalanceWouldGoNegative = errors.New("balance would go negative")
	ErrBalanceOverflow        = errors.New("balance overflow")
)
