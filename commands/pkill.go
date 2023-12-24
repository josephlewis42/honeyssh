package commands

import (
	"github.com/josephlewis42/honeyssh/core/vos"
)

// Pkill implements a no-op pkill command.
func Pkill(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "pkill [OPTION]... PATTERN",
		Short: "Signal a process by pattern",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		// No-op.
		return 0
	})
}

var _ vos.ProcessFunc = Pkill

func init() {
	addBinCmd("pkill", Pkill)
}
