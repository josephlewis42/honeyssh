package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
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
		host, err := virtOS.Hostname()
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "hostname: %v\n", err)
			return 1
		}

		fmt.Fprintln(virtOS.Stdout(), host)
		return 0
	})
}

var _ HoneypotCommandFunc = Hostname

func init() {
	addBinCmd("hostname", Hostname)
}
