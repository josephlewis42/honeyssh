package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Clear implements the UNIX clear command.
func Clear(virtOS vos.VOS) int {
	if virtOS.GetPTY().IsPTY {
		// Assumes VT100 compatibility.
		fmt.Fprintf(virtOS.Stdout(), "\033[0;0H")
	}
	return 0
}

var _ HoneypotCommandFunc = Clear

func init() {
	addBinCmd("clear", HoneypotCommandFunc(Clear))
}
