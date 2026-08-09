package httpapi

import (
	"strings"
	"testing"
)

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
