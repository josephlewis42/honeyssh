package commands

import (
	"fmt"
	"text/tabwriter"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// W implements the UNIX w command.
func W(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "w",
		Short: "Show who is logged in and what they're doing.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), formatUptime(virtOS))

		w := tabwriter.NewWriter(virtOS.Stdout(), 0, 8, 2, ' ', 0)
		defer w.Flush()
		fmt.Fprintln(w, "USER\tTTY\tFROM\tLOGIN@\tIDLE\tJCPU\tCPU\tWHAT")
		fmt.Fprintf(w, "%s\tpts/0\t%s\t%s\t0.00s\t0.00s\t0.00s\tw\n",
			virtOS.SSHUser(),
			virtOS.SSHRemoteAddr(),
			virtOS.LoginTime().Format("15:04"),
		)

		return 0
	})
}

var _ vos.ProcessFunc = W

func init() {
	mustAddBinCmd("w", W)
	mustAddBinCmd("who", W)
}
