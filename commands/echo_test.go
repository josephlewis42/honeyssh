package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnescape(t *testing.T) {
	cases := []struct {
		escaped  string
		expected string
	}{
		{"not escaped", "not escaped"},
		{`newline\n`, "newline\n"},
		{`double-escape\\n`, `double-escape\n`},
		{`double-escape\\n`, `double-escape\n`},
		// Octal
		{`\07`, string(rune(7))},
		{`\011`, "\t"},
		{`\0101`, "A"},
		// Hex
		{`\x7`, string(rune(07))},
		{`\x9`, "\t"},
		{`\x4A`, "J"},
	}

	for _, tc := range cases {
		t.Run(tc.escaped, func(t *testing.T) {
			actual := unescape(tc.escaped)

			assert.Equal(t, tc.expected, actual)
		})
	}
}
