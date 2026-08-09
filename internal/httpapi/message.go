package httpapi

import (
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
