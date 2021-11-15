package commands

import (
	"flag"
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Which implements the UNIX which command.
func Which(virtOS vos.VOS) int {
	flags := flag.NewFlagSet("which", flag.ContinueOnError)
	flags.SetOutput(virtOS.Stderr())
	if err := flags.Parse(virtOS.Args()[1:]); err != nil {
		fmt.Fprintln(virtOS.Stderr(), "Usage: which args")
		fmt.Fprintln(virtOS.Stderr(), "Locate a command.")
		return 1
	}

	for _, arg := range flags.Args() {
		res, err := vos.LookPath(virtOS, arg)
		if err == nil {
			fmt.Fprintln(virtOS.Stdout(), res)
		} else {
			fmt.Fprintln(virtOS.Stderr(), err)
		}
	}

	return 0
}

var _ HoneypotCommandFunc = Which

func init() {
	addBinCmd("which", HoneypotCommandFunc(Which))
}
