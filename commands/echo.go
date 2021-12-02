package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"josephlewis.net/osshit/core/vos"
)

var (
	unescapeOctal   = regexp.MustCompile(`\\0[0-8][0-8]?[0-8]?`)
	unescapeHex     = regexp.MustCompile(`\\x[0-9a-fA-F][0-9a-fA-F]?`)
	unescapeReplace = strings.NewReplacer(
		`\n`, "\n", // newline
		`\r`, "\r", // carriage return
		`\t`, "\t", // horizontal tab
		`\\`, `\`, // backslash literal
		`\b`, "\b", // backspace
		`\a`, "\a", // alert
		`\f`, "\f", // form feed
		`\v`, "\v", // vertical tab
	)
)

func unescape(s string) string {
	s = unescapeReplace.Replace(s)
	s = unescapeOctal.ReplaceAllStringFunc(s, func(arg string) string {
		out, err := strconv.ParseInt(arg[2:], 8, 8)
		if err != nil {
			return arg
		}
		return string(rune(out))
	})
	s = unescapeHex.ReplaceAllStringFunc(s, func(arg string) string {
		out, err := strconv.ParseInt(arg[2:], 16, 8)
		if err != nil {
			return arg
		}
		return string(rune(out))
	})
	return s
}

// Echo implements a limited echo command.
func Echo(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "echo [-e] [ARG] ...",
		Short: "Display a line of text.",
	}

	opt := cmd.Flags()
	escaped := opt.Bool('e', "interpret backslash escapes")

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		for i, arg := range opt.Args() {
			if i > 0 {
				fmt.Fprint(w, " ")
			}

			if *escaped {
				arg = unescape(arg)
			}

			fmt.Fprint(w, arg)
		}

		fmt.Fprintln(w)

		return 0
	})
}

var _ HoneypotCommandFunc = Echo

func init() {
	addBinCmd("echo", Echo)
}
