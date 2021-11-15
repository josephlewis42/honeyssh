package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Reset implements the UNIX reset command.
func Reset(virtOS vos.VOS) int {
	if virtOS.GetPTY().IsPTY {
		// Assumes VT100 compatibility.
		fmt.Fprintf(virtOS.Stdout(), "\033c")
	}
	return 0
}

var _ HoneypotCommandFunc = Reset

func init() {
	addBinCmd("reset", HoneypotCommandFunc(Reset))
}
