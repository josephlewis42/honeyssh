package commands

import (
	"fmt"

	getopt "github.com/pborman/getopt/v2"
	"josephlewis.net/osshit/core/vos"
)

// Uname implements the POSIX command by the same name.
func Uname(virtOS vos.VOS) int {
	opts := getopt.New()

	showAll := opts.BoolLong("all", 'a', "print all information")
	showKernelName := opts.BoolLong("kernel-name", 's', "print the kernel name")
	showNodename := opts.BoolLong("nodename", 'n', "print the network node name")
	showRelease := opts.BoolLong("kernel-release", 'r', "print the kernel release")
	showVersion := opts.BoolLong("kernel-version", 'v', "print the kernel version")
	showMachine := opts.BoolLong("machine", 'm', "print the machine name")
	showHelp := opts.BoolLong("help", 'h', "show help")

	w := virtOS.Stdout()
	if err := opts.Getopt(virtOS.Args(), nil); err != nil || *showHelp {
		if err != nil {
			virtOS.LogInvalidInvocation(err)
			fmt.Fprintf(virtOS.Stderr(), "error: %s\n\n", err)
		}
		fmt.Fprintln(w, "usage: uname [OPTIONS...]")
		fmt.Fprintln(w, "Display system informamtion.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		opts.PrintOptions(w)

		return 1
	}

	uname := virtOS.Uname()
	anyPrinted := false
	for _, entry := range []struct {
		flag     *bool
		property string
	}{
		{showKernelName, uname.Sysname},
		{showNodename, uname.Nodename},
		{showRelease, uname.Release},
		{showVersion, uname.Version},
		{showMachine, uname.Machine},
	} {
		if *entry.flag || *showAll {
			if anyPrinted {
				fmt.Fprintf(w, " ")
			}
			fmt.Fprintf(w, "%s", entry.property)
			anyPrinted = true
		}
	}

	if !anyPrinted {
		fmt.Fprintf(w, uname.Sysname)
	}

	fmt.Fprintln(w)

	return 0
}

var _ HoneypotCommandFunc = Uname

func init() {
	addBinCmd("uname", HoneypotCommandFunc(Uname))
}
