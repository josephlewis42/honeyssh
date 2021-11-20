package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Mail implements a fake mail command.
func Mail(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "mail [OPTION...] [address...]",
		Short: "process mail messages",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintf(virtOS.Stdout(), "No mail for %s\n", virtOS.SSHUser())
		return 0
	})
}

var _ HoneypotCommandFunc = Mail

func init() {
	addBinCmd("mail", HoneypotCommandFunc(Mail))
}
