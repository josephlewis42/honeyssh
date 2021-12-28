package commands

import (
	"fmt"
	"sort"

	"github.com/josephlewis42/honeyssh/core/vos"
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
		env := virtOS.Environ()
		sort.Strings(env)
		for _, envDef := range env {
			fmt.Fprintln(virtOS.Stdout(), envDef)
		}

		return 0
	})
}

var _ vos.ProcessFunc = Env

func init() {
	addBinCmd("env", Env)
}
