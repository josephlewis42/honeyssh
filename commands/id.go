package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Id implements a the POSIX id command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/id.html
func Id(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "id [OPTION]... [USER]",
		Short: "Get the user's identity.",
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
	addBinCmd("id", Id)
}
