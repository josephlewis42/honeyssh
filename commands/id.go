package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Id implements a fake id command.
func Id(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "id [OPTION]... [USER]",
		Short: "Print user and group information.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintf(w, "uid=%[1]d(%[2]s) gid=%[1]d(%[2]s) groups=%[1]d(%[2]s)\n", virtOS.Getuid(), virtOS.SSHUser())
		return 0
	})
}

var _ HoneypotCommandFunc = Id

func init() {
	addBinCmd("id", HoneypotCommandFunc(Id))
}
