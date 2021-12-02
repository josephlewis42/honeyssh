package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Mail implements a no-op POSIX mail command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/mailx.html
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
