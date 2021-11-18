package commands

import (
	"fmt"

	getopt "github.com/pborman/getopt/v2"
	"josephlewis.net/osshit/core/vos"
)

// Hostname implements the POSIX command by the same name.
func Hostname(virtOS vos.VOS) int {
	opts := getopt.New()

	if err := opts.Getopt(virtOS.Args(), nil); err != nil {
		virtOS.LogInvalidInvocation(err)
	}

	host, err := virtOS.Hostname()
	if err != nil {
		fmt.Fprintf(virtOS.Stderr(), "error: %v\n", err)
		return 1
	}

	fmt.Fprintln(virtOS.Stdout(), host)
	return 0
}

var _ HoneypotCommandFunc = Hostname

func init() {
	addBinCmd("hostname", HoneypotCommandFunc(Hostname))
}
