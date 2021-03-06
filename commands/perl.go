package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Perl implements a fake Perl interpreter.
func Perl(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "perl [switches] [--] [programfile] [arguments]",
		Short: "The Perl 5 language interpreter.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintln(w, `Can't locate perl5db.pl: No such file or directory`)
		return 1
	})
}

var _ vos.ProcessFunc = Perl

func init() {
	addBinCmd("perl", Perl)
}
