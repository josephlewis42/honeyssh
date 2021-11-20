package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Segfault fails.
func Segfault(virtOS vos.VOS) int {
	name := virtOS.Args()[0]
	fmt.Fprintf(virtOS.Stdout(), "%s: Segmentation fault\n", name)

	return 1
}

var SegfaultCommand HoneypotCommandFunc = Segfault
