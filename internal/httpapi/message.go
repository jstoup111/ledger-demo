package httpapi

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

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
