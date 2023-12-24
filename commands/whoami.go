package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Whoami implements the POSIX whoami command.
func Whoami(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "whoami [OPTION]...",
		Short: "Print the current user.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintln(w, virtOS.SSHUser())
		return 0
	})
}

var _ vos.ProcessFunc = Whoami

func init() {
	mustAddBinCmd("whoami", Whoami)
}
