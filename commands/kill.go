package commands

import (
	"josephlewis.net/osshit/core/vos"
)

// Kill implements a no-op kill command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/
func Kill(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "kill [-s sigspec | -n signum | -sigspec] pid | jobspec ... or kill -l [sigspec]",
		Short: "Send a signal to a process.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// No-op.
		return 0
	})
}

var _ HoneypotCommandFunc = Kill

func init() {
	addBinCmd("kill", Kill)
}
