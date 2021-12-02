package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Env implements the POSIX env command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/env.html
func Env(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "env",
		Short: "Set or print the environment for command invocation.",
	}

	return cmd.Run(virtOS, func() int {
		for _, envDef := range virtOS.Environ() {
			fmt.Fprintln(virtOS.Stdout(), envDef)
		}

		return 0
	})
}

var _ HoneypotCommandFunc = Env

func init() {
	addBinCmd("env", Env)
}
