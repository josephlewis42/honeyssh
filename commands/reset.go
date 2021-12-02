package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Reset sends an ANSI reset command if connected to a PTY.
func Reset(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "reset",
		Short: "Sets the terminal modes to default values.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		if virtOS.GetPTY().IsPTY {
			// Assumes VT100 compatibility.
			fmt.Fprintf(virtOS.Stdout(), "\033c")
		}
		return 0
	})
}

var _ HoneypotCommandFunc = Reset

func init() {
	addBinCmd("reset", Reset)
}
