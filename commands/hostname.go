package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Hostname implements the Linux command by the same name.
func Hostname(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "hostname [hostname]",
		Short: "Get or set the system's hostname.",
		// Never bail, even if flags are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), virtOS.Hostname())
		return 0
	})
}

var _ vos.ProcessFunc = Hostname

func init() {
	mustAddBinCmd("hostname", Hostname)
}
