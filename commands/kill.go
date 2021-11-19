package commands

import (
	"josephlewis.net/osshit/core/vos"
)

// Kill implements a fake kill command.
func Kill(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "kill [-s sigspec | -n signum | -sigspec] pid | jobspec ... or kill -l [sigspec]",
		Short: "Send a signal to a job.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// Noop
		return 0
	})
}

var _ HoneypotCommandFunc = Kill

func init() {
	addBinCmd("kill", HoneypotCommandFunc(Kill))
}
