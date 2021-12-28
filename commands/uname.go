package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Uname implements the POSIX command by the same name.
func Uname(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "uname [OPTIONS...]",
		Short: "Display system informamtion.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	opts := cmd.Flags()
	showAll := opts.BoolLong("all", 'a', "print all information")
	showKernelName := opts.BoolLong("kernel-name", 's', "print the kernel name")
	showNodename := opts.BoolLong("nodename", 'n', "print the network node name")
	showRelease := opts.BoolLong("kernel-release", 'r', "print the kernel release")
	showVersion := opts.BoolLong("kernel-version", 'v', "print the kernel version")
	showMachine := opts.BoolLong("machine", 'm', "print the machine name")

	return cmd.Run(virtOS, func() int {

		w := virtOS.Stdout()
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
	})
}

var _ vos.ProcessFunc = Uname

func init() {
	addBinCmd("uname", Uname)
}
