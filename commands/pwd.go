package commands

import (
	"flag"
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Pwd implements the UNIX pwd command.
func Pwd(virtOS vos.VOS) int {
	flags := flag.NewFlagSet("pwd", flag.ContinueOnError)
	flags.SetOutput(virtOS.Stderr())
	if err := flags.Parse(virtOS.Args()[1:]); err != nil {
		virtOS.LogInvalidInvocation(err)

		fmt.Fprintln(virtOS.Stderr(), "Usage: pwd")
		fmt.Fprintln(virtOS.Stderr(), "Print the name of the current working directory.")
		return 1
	}

	pwd, err := virtOS.Getwd()
	if err != nil {
		fmt.Fprintf(virtOS.Stderr(), "%v\n", err)
		return 1
	}

	fmt.Fprintln(virtOS.Stdout(), pwd)

	return 0
}

var _ HoneypotCommandFunc = Pwd

func init() {
	addBinCmd("pwd", Pwd)
}
