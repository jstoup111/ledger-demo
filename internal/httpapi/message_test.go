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

func TestMessageForIncludesRequestedAccountOnlyForAccountNotFound(t *testing.T) {
	tests := []struct {
		name    string
		context messageContext
		want    string
	}{
		{
			name:    "includes requested account ID",
			context: messageContext{accountID: "acct-9"},
			want:    "Account not found. Requested: acct-9.",
		},
		{
			name:    "omits empty requested account ID",
			context: messageContext{},
			want:    "Account not found.",
		},
		{
			name:    "omits over-long requested account ID",
			context: messageContext{accountID: strings.Repeat("a", 33)},
			want:    "Account not found.",
		},
		{
			name:    "omits control-character-bearing requested account ID",
			context: messageContext{accountID: "acct\nnope"},
			want:    "Account not found.",
		},
		{
			name:    "ignores carried value",
			context: messageContext{value: "ignored"},
			want:    "Account not found.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageFor("account_not_found", tt.context); got != tt.want {
				t.Errorf("messageFor(account_not_found, %+v) = %q, want %q", tt.context, got, tt.want)
			}
		})
	}
}

func TestMessageForNamesSubmittedAmountForAmountRejections(t *testing.T) {
	tests := []struct {
		name         string
		identifier   string
		value        string
		want         string
		rejectsValue bool
	}{
		{
			name:       "zero amount",
			identifier: "amount_zero",
			value:      "0.00",
			want:       "Amount must not be zero. Submitted: 0.00.",
		},
		{
			name:       "malformed amount",
			identifier: "amount_malformed",
			value:      "12.3.4",
			want:       "Amount is malformed. Submitted: 12.3.4.",
		},
		{
			name:         "zero amount with empty submitted value",
			identifier:   "amount_zero",
			value:        "",
			want:         "Amount must not be zero.",
			rejectsValue: true,
		},
		{
			name:         "zero amount with thirty three rune submitted value",
			identifier:   "amount_zero",
			value:        strings.Repeat("a", 33),
			want:         "Amount must not be zero.",
			rejectsValue: true,
		},
		{
			name:         "zero amount with newline submitted value",
			identifier:   "amount_zero",
			value:        "0.00\nnext",
			want:         "Amount must not be zero.",
			rejectsValue: true,
		},
		{
			name:         "malformed amount with empty submitted value",
			identifier:   "amount_malformed",
			value:        "",
			want:         "Amount is malformed.",
			rejectsValue: true,
		},
		{
			name:         "malformed amount with thirty three rune submitted value",
			identifier:   "amount_malformed",
			value:        strings.Repeat("a", 33),
			want:         "Amount is malformed.",
			rejectsValue: true,
		},
		{
			name:         "malformed amount with newline submitted value",
			identifier:   "amount_malformed",
			value:        "12.3.4\nnext",
			want:         "Amount is malformed.",
			rejectsValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messageFor(tt.identifier, messageContext{value: tt.value})
			if got != tt.want {
				t.Errorf("messageFor(%q, value %q) = %q, want %q", tt.identifier, tt.value, got, tt.want)
			}
			if tt.rejectsValue && tt.value != "" && strings.Contains(got, tt.value) {
				t.Errorf("messageFor(%q, value %q) included rejected value in %q", tt.identifier, tt.value, got)
			}
		})
	}
}

func TestMessageForNamesPostingAndBalanceForBalanceRejections(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		context    messageContext
		want       string
	}{
		{
			name:       "negative balance with known balance",
			identifier: "balance_would_go_negative",
			context: messageContext{
				value:        "-200000",
				balance:      128350,
				balanceKnown: true,
			},
			want: "Balance would go negative. Posting -$2,000.00 against a balance of $1,283.50.",
		},
		{
			name:       "overflow with known balance",
			identifier: "balance_overflow",
			context: messageContext{
				value:        "-200000",
				balance:      128350,
				balanceKnown: true,
			},
			want: "Balance would overflow.",
		},
		{
			name:       "negative balance with unknown balance",
			identifier: "balance_would_go_negative",
			context:    messageContext{value: "-200000"},
			want:       "Balance would go negative.",
		},
		{
			name:       "overflow with unknown balance",
			identifier: "balance_overflow",
			context:    messageContext{value: "-200000"},
			want:       "Balance would overflow.",
		},
		{
			name:       "negative balance with invalid cents",
			identifier: "balance_would_go_negative",
			context: messageContext{
				value:        "12.50",
				balance:      128350,
				balanceKnown: true,
			},
			want: "Balance would go negative.",
		},
		{
			name:       "overflow with invalid cents",
			identifier: "balance_overflow",
			context: messageContext{
				value:        "12.50",
				balance:      128350,
				balanceKnown: true,
			},
			want: "Balance would overflow.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := messageFor(tt.identifier, tt.context); got != tt.want {
				t.Errorf("messageFor(%q, %+v) = %q, want %q", tt.identifier, tt.context, got, tt.want)
			}
		})
	}
}

func TestMessageForNamesSubmittedCharacterCountForDescriptionTooLong(t *testing.T) {
	context := messageContext{value: "187"}
	want := "Description is too long. Submitted: 187 characters; the limit is 140."

	if got := messageFor("description_too_long", context); got != want {
		t.Errorf("messageFor(description_too_long, %+v) = %q, want %q", context, got, want)
	}
}

func TestMessageForIgnoresCarriedValuesWithoutMeaning(t *testing.T) {
	values := []struct {
		name  string
		value string
	}{
		{name: "zero amount", value: "0.00"},
		{name: "script text", value: "<script>alert(1)</script>"},
		{name: "five hundred characters", value: strings.Repeat("a", 500)},
	}
	identifiers := []struct {
		name       string
		identifier string
		want       string
	}{
		{name: "description required", identifier: "description_empty", want: "Description must not be empty."},
		{name: "first unknown identifier", identifier: "not_a_real_code", want: "Unable to post transaction."},
		{name: "second unknown identifier", identifier: "another_unknown_code", want: "Unable to post transaction."},
	}

	for _, identifier := range identifiers {
		for _, value := range values {
			t.Run(identifier.name+" with "+value.name, func(t *testing.T) {
				got := messageFor(identifier.identifier, messageContext{value: value.value})
				if got != identifier.want || strings.Contains(got, value.value) {
					t.Errorf("messageFor(%q, value %q) = %q, want %q without the carried value", identifier.identifier, value.value, got, identifier.want)
				}
			})
		}
	}
}

func TestMessageForOmitsRejectedDescriptionCharacterCounts(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "letters", value: "abc"},
		{name: "below limit", value: "3"},
		{name: "at limit", value: "140"},
		{name: "empty", value: ""},
		{name: "seven digits", value: "1000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messageFor("description_too_long", messageContext{value: tt.value})
			if got != "Description is too long." {
				t.Errorf("messageFor(description_too_long, value %q) = %q, want plain sentence", tt.value, got)
			}
			if tt.value != "" && strings.Contains(got, tt.value) {
				t.Errorf("messageFor(description_too_long, value %q) included rejected value in %q", tt.value, got)
			}
		})
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
