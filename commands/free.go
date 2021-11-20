package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Free implements a fake free command.
func Free(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "free [OPTION]...",
		Short: "Display amount of free and used memory in the system.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	humanSize := cmd.Flags().BoolLong("human-readable", 'h', "print human readable sizes")
	cmd.ShowHelp = cmd.Flags().BoolLong("help", '?', "show help and exit")

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()

		if *humanSize {
			fmt.Fprintln(w,
				`              total        used        free      shared  buff/cache   available
Mem:           7.2G        4.1G        1.8G        623M        1.3G        2.2G
Swap:           23G        4.1G         19G`)

		} else {
			fmt.Fprintln(w,
				`              total        used        free      shared  buff/cache   available
Mem:        7596572     4387812     1215572      730612     1993188     2171992
Swap:      24587768     4301240    20286528`)
		}
		return 0
	})
}

var _ HoneypotCommandFunc = Free

func init() {
	addBinCmd("free", HoneypotCommandFunc(Free))
}
