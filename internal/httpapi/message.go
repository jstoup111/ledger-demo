package httpapi

import (
	"errors"
	"math"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/jstoup111/ledger-demo/internal/ledger"
)

type messageContext struct {
	value        string
	accountID    string
	balance      int64
	balanceKnown bool
}

func messageFor(identifier string, context messageContext) string {
	switch identifier {
	case "":
		return ""
	case "account_not_found":
		if accountID, ok := freeTextCarriedValue(context.accountID); ok {
			return "Account not found. Requested: " + accountID + "."
		}
		return "Account not found."
	case "amount_zero":
		if value, ok := zeroAmountCarriedValue(context.value); ok {
			return "Amount must not be zero. Submitted: " + value + "."
		}
		return "Amount must not be zero."
	case "description_empty":
		return "Description must not be empty."
	case "description_too_long":
		if value, ok := characterCountCarriedValue(context.value); ok {
			return "Description is too long. Submitted: " + value + " characters; the limit is 140."
		}
		return "Description is too long."
	case "amount_malformed":
		if value, ok := malformedAmountCarriedValue(context.value); ok {
			return "Amount is malformed. Submitted: " + value + "."
		}
		return "Amount is malformed."
	case "balance_would_go_negative":
		return balanceRejectionMessage(identifier, "Balance would go negative.", context)
	case "balance_overflow":
		return balanceRejectionMessage(identifier, "Balance would overflow.", context)
	default:
		return "Unable to post transaction."
	}
}

func balanceRejectionMessage(identifier, message string, context messageContext) string {
	value, ok := integerCentsCarriedValue(context.value)
	if !ok || !context.balanceKnown {
		return message
	}

	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return message
	}

	switch identifier {
	case "balance_would_go_negative":
		if !(context.balance >= 0 && cents < 0 && cents < -context.balance) {
			return message
		}
	case "balance_overflow":
		if cents <= 0 || context.balance < 0 || cents <= math.MaxInt64-context.balance {
			return message
		}
	default:
		return message
	}

	return message + " Posting " + formatDollars(cents) + " against a balance of " + formatDollars(context.balance) + "."
}

func freeTextCarriedValue(value string) (string, bool) {
	if value == "" || utf8.RuneCountInString(value) > 32 {
		return "", false
	}

	hasNonSpaceRune := false
	for _, r := range value {
		if !unicode.IsGraphic(r) {
			return "", false
		}
		if !unicode.IsSpace(r) {
			hasNonSpaceRune = true
		}
	}

	if !hasNonSpaceRune {
		return "", false
	}

	return value, true
}

func zeroAmountCarriedValue(value string) (string, bool) {
	value, ok := freeTextCarriedValue(value)
	if !ok {
		return "", false
	}

	_, err := parseAmount(value)
	return value, errors.Is(err, ledger.ErrAmountZero)
}

func malformedAmountCarriedValue(value string) (string, bool) {
	value, ok := freeTextCarriedValue(value)
	if !ok {
		return "", false
	}

	_, err := parseAmount(value)
	return value, errors.Is(err, ledger.ErrAmountMalformed)
}

func characterCountCarriedValue(value string) (string, bool) {
	if len(value) == 0 || len(value) > 6 || value[0] == '0' {
		return "", false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return "", false
		}
	}

	count, err := strconv.Atoi(value)
	if err != nil || count <= 140 {
		return "", false
	}

	return value, true
}

func integerCentsCarriedValue(value string) (string, bool) {
	if value == "" || len(value) > 20 {
		return "", false
	}

	digits := value
	if value[0] == '-' {
		if len(value) == 1 {
			return "", false
		}
		digits = value[1:]
	}

	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", false
		}
	}

	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return "", false
	}

	return value, true
}
