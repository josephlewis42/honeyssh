package commands

import (
	"josephlewis.net/osshit/core/vos"
)

// Nice implements a fake kill command.
func Nice(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "nice [OPTION] [COMMAND [ARG]...]",
		Short: "Run command with adjusted niceness.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// Noop
		return 0
	})
}

var _ HoneypotCommandFunc = Nice

func init() {
	addBinCmd("nice", HoneypotCommandFunc(Nice))
}
