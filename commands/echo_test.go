package commands

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"josephlewis.net/osshit/core/vos/vostest"
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

func ExampleEcho_help() {
	cmd := vostest.Command(Echo, "echo", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	// Output: usage: echo [-e] [ARG] ...
	// Display a line of text.
	//
	// Flags:
	//  -e          interpret backslash escapes
	//  -h, --help  show this help and exit
}

func ExampleEcho_echo() {
	cmd := vostest.Command(Echo, "echo", "Hello, world!")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	// Output: Hello, world!
}

func ExampleEcho_echoEscape() {
	cmd := vostest.Command(Echo, "echo", "-e", `Hello\nworld!`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	// Output: Hello
	// world!
}

func ExampleEcho_echoMultiple() {
	cmd := vostest.Command(Echo, "echo", "a", "b", "c")
	out, err := cmd.CombinedOutput()
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	// Output: a b c
}
