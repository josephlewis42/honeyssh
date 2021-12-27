package commands

import (
	"fmt"

	"josephlewis.net/honeyssh/core/vos"
)

// Make implements a no-op POSIX make command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/make.html
func Make(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "make [options] [target] ...",
		Short: "Run a dependency graph of commands.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stderr(), "make: *** No rule to make target. Stop.")
		return 1
	})
}

var _ vos.ProcessFunc = Make

func init() {
	addBinCmd("make", Make)
}
