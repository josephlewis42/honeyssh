package commands

import (
	"github.com/josephlewis42/honeyssh/core/vos"
)

// Killall implements a no-op killall command.
func Killall(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "killall [OPTION]... [--] NAME...",
		Short: "Kill a process by name.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// No-op.
		return 0
	})
}

var _ vos.ProcessFunc = Killall

func init() {
	addBinCmd("killall", Killall)
}
