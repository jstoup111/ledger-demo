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
