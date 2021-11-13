package commands

import (
	"flag"
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Env implements the UNIX env command.
func Env(virtOS vos.VOS) int {
	flags := flag.NewFlagSet("env", flag.ContinueOnError)
	flags.SetOutput(virtOS.Stderr())
	if err := flags.Parse(virtOS.Args()[1:]); err != nil {
		fmt.Fprintln(virtOS.Stderr(), "Usage: env")
		fmt.Fprintln(virtOS.Stderr(), "Print the resulting environment.")
		return 1
	}

	for _, envDef := range virtOS.Environ() {
		fmt.Fprintln(virtOS.Stdout(), envDef)
	}

	return 0
}

var _ HoneypotCommandFunc = Env

func init() {
	addBinCmd("env", HoneypotCommandFunc(Env))
}
