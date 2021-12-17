package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Nice implements a fake POSIX nice command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/nice.html
func Nice(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "nice [OPTION] [COMMAND [ARG]...]",
		Short: "Run command with adjusted niceness.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	_ = cmd.Flags().IntLong("niceness", 'n', 10, "Amount to add to the niceness.")

	return cmd.Run(virtOS, func() int {
		args := cmd.Flags().Args()

		if len(args) == 0 {
			fmt.Fprintln(virtOS.Stdout(), "0")
			return 0
		}

		proc, err := virtOS.StartProcess(args[0], args, &vos.ProcAttr{
			Files: virtOS,
		})
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "nice: couldn't start process: %v\n", err)
			return 1
		}

		return proc.Run()
	})
}

var _ vos.ProcessFunc = Nice

func init() {
	addBinCmd("nice", Nice)
}
