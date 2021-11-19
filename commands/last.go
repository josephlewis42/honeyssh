package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Last implements a fake last command.
func Last(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "last [options] ...",
		Short: "Show a listing of last logged in users.",
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintf(w,
			"%s    %s    %s\n",
			virtOS.SSHUser(),
			"pts/0",
			virtOS.LoginTime().Format("Mon Jan _2 15:04"),
		)

		fmt.Fprintln(w)
		fmt.Fprintf(w, "wtmp begins %s\n", virtOS.BootTime().Format("Mon Jan _2 15:04 2006"))
		return 0
	})
}

var _ HoneypotCommandFunc = Last

func init() {
	addBinCmd("last", HoneypotCommandFunc(Last))
}
