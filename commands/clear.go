package commands

import (
	"fmt"

	"josephlewis.net/honeyssh/core/vos"
)

// Clear sends an ANSI clear command if connected to a PTY.
func Clear(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "clear",
		Short: "Clears the screen.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		if virtOS.GetPTY().IsPTY {
			// Assumes VT100 compatibility.
			fmt.Fprintf(virtOS.Stdout(), "\033[0;0H")
		}
		return 0
	})
}

var _ vos.ProcessFunc = Clear

func init() {
	addBinCmd("clear", Clear)
}
