package commands

import (
	"github.com/josephlewis42/honeyssh/core/vos"
)

// Screen implements a fake screen command.
func Screen(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "screen [-opts] [cmd [args]]",
		Short: "screen manager with VT100/ANSI terminal emulation",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// Noop
		return 0
	})
}

var _ vos.ProcessFunc = Screen

func init() {
	addBinCmd("screen", Screen)
}
