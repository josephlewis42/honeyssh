package commands

import (
	"fmt"

	"josephlewis.net/honeyssh/core/vos"
)

// Python implements a fake Python interpreter.
func Python(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "python [option] ... [-c cmd | -m mod | file | -] [arg] ...",
		Short: "Embedded version of the Python language.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintln(w, "python: No module named os")
		return 1
	})
}

var _ vos.ProcessFunc = Python

func init() {
	addBinCmd("python", Python)
}
