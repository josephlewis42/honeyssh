package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Reboot terminates the remote connection.
func Reboot(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "reboot [options] [arg] ...",
		Short: "Reboot the system.",
	}

	return cmd.Run(virtOS, func() int {
		// Broadcast to SSHStdout to bypass others like `wall` would do
		fmt.Fprintf(virtOS.SSHStdout(), "Broadcast message from root@%s:\n", virtOS.Hostname())
		fmt.Fprintln(virtOS.SSHStdout(), "The system is going down for reboot NOW!")
		virtOS.SSHExit(0)
		return 0
	})
}

var _ vos.ProcessFunc = Reboot

func init() {
	mustAddSbinCmd("reboot", Reboot)
}
