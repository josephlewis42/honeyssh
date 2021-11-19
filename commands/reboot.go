package commands

import (
	"fmt"

	"josephlewis.net/osshit/core/vos"
)

// Reboot terminates the remote connection.
func Reboot(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "reboot [options] [arg] ...",
		Short: "Reboot the system.",
	}

	return cmd.Run(virtOS, func() int {
		// Broadcast to SSHStdout to bypass others like `wall` would do
		host, _ := virtOS.Hostname()
		fmt.Fprintf(virtOS.SSHStdout(), "Broadcast message from root@%s:\n", host)
		fmt.Fprintln(virtOS.SSHStdout(), "The system is going down for reboot NOW!")
		virtOS.SSHExit(0)
		return 0
	})
}

var _ HoneypotCommandFunc = Reboot

func init() {
	addSbinCmd("reboot", HoneypotCommandFunc(Reboot))
}
