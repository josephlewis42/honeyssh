package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Make implements a fake make command.
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

var _ HoneypotCommandFunc = Make

func init() {
	addBinCmd("make", HoneypotCommandFunc(Make))
}
