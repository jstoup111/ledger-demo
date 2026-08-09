package httpapi

import (
	"strings"
	"testing"
)

func TestMessageForReturnsPlainSentencesAndGenericFallback(t *testing.T) {
	context := messageContext{}
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "account not found", identifier: "account_not_found", want: "Account not found."},
		{name: "amount zero", identifier: "amount_zero", want: "Amount must not be zero."},
		{name: "description required", identifier: "description_empty", want: "Description must not be empty."},
		{name: "description too long", identifier: "description_too_long", want: "Description is too long."},
		{name: "amount malformed", identifier: "amount_malformed", want: "Amount is malformed."},
		{name: "balance would go negative", identifier: "balance_would_go_negative", want: "Balance would go negative."},
		{name: "balance overflow", identifier: "balance_overflow", want: "Balance would overflow."},
		{name: "unknown identifier", identifier: "not_a_real_code", want: "Unable to post transaction."},
		{name: "empty identifier", identifier: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageFor(tt.identifier, context); got != tt.want {
				t.Errorf("messageFor(%q, zero context) = %q, want %q", tt.identifier, got, tt.want)
			}
		})
	}

	firstUnknown := messageFor("not_a_real_code", context)
	secondUnknown := messageFor("another_unknown_code", context)
	if firstUnknown != secondUnknown {
		t.Errorf("different unknown identifiers returned different messages: %q and %q", firstUnknown, secondUnknown)
	}
}

func TestFreeTextCarriedValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "zero amount", input: "0.00", want: true},
		{name: "malformed amount", input: "12.3.4", want: true},
		{name: "letters", input: "abc", want: true},
		{name: "thirty two runes", input: strings.Repeat("a", 32), want: true},
		{name: "thirty two multibyte runes", input: strings.Repeat("é", 32), want: true},
		{name: "script text", input: "<script>alert(1)</script>", want: true},
		{name: "empty", input: "", want: false},
		{name: "thirty three runes", input: strings.Repeat("a", 33), want: false},
		{name: "newline", input: "value\nnext", want: false},
		{name: "carriage return", input: "value\rnext", want: false},
		{name: "tab", input: "value\tnext", want: false},
		{name: "nul", input: "value\x00next", want: false},
		{name: "next line", input: "value\u0085next", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := freeTextCarriedValue(tt.input)
			if ok != tt.want {
				t.Fatalf("freeTextCarriedValue(%q) ok = %t, want %t", tt.input, ok, tt.want)
			}
			if ok && got != tt.input {
				t.Errorf("freeTextCarriedValue(%q) = %q, want original value", tt.input, got)
			}
		})
	}
}

func TestCharacterCountCarriedValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "one hundred forty one characters", input: "141", want: true},
		{name: "one hundred eighty seven characters", input: "187", want: true},
		{name: "six digit character count", input: "999999", want: true},
		{name: "empty", input: "", want: false},
		{name: "letters", input: "abc", want: false},
		{name: "below limit", input: "3", want: false},
		{name: "at limit", input: "140", want: false},
		{name: "negative", input: "-5", want: false},
		{name: "decimal", input: "1.5", want: false},
		{name: "seven digit value", input: "1000000", want: false},
		{name: "thirty digit value", input: "123456789012345678901234567890", want: false},
		{name: "leading zero", input: "0141", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := characterCountCarriedValue(tt.input)
			if ok != tt.want {
				t.Fatalf("characterCountCarriedValue(%q) ok = %t, want %t", tt.input, ok, tt.want)
			}
		})
	}
}

func TestIntegerCentsCarriedValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "one cent", input: "1", want: true},
		{name: "negative one cent", input: "-1", want: true},
		{name: "two thousand dollars", input: "200000", want: true},
		{name: "negative two thousand dollars", input: "-200000", want: true},
		{name: "maximum int64", input: "9223372036854775807", want: true},
		{name: "minimum int64", input: "-9223372036854775808", want: true},
		{name: "empty", input: "", want: false},
		{name: "letters", input: "abc", want: false},
		{name: "decimal", input: "12.50", want: false},
		{name: "exponent", input: "1e9", want: false},
		{name: "plus sign", input: "+5", want: false},
		{name: "minus sign only", input: "-", want: false},
		{name: "above maximum int64", input: "9223372036854775808", want: false},
		{name: "forty digits", input: strings.Repeat("1", 40), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := integerCentsCarriedValue(tt.input)
			if ok != tt.want {
				t.Fatalf("integerCentsCarriedValue(%q) ok = %t, want %t", tt.input, ok, tt.want)
			}
			if ok && got != tt.input {
				t.Errorf("integerCentsCarriedValue(%q) = %q, want original value", tt.input, got)
			}
		})
	}
}
