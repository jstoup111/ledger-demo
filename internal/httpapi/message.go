package httpapi

import (
	"strconv"
	"unicode"
	"unicode/utf8"
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
		if context.accountID != "" {
			return "Account not found. Requested: " + context.accountID + "."
		}
		return "Account not found."
	case "amount_zero":
		return "Amount must not be zero."
	case "description_empty":
		return "Description must not be empty."
	case "description_too_long":
		return "Description is too long."
	case "amount_malformed":
		return "Amount is malformed."
	case "balance_would_go_negative":
		return "Balance would go negative."
	case "balance_overflow":
		return "Balance would overflow."
	default:
		return "Unable to post transaction."
	}
}

func freeTextCarriedValue(value string) (string, bool) {
	if value == "" || utf8.RuneCountInString(value) > 32 {
		return "", false
	}

	for _, r := range value {
		if unicode.IsControl(r) {
			return "", false
		}
	}

	return value, true
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
