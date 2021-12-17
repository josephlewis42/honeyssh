package commands

import (
	"fmt"
	"text/tabwriter"

	"josephlewis.net/osshit/core/vos"
)

// W implements the UNIX w command.
func W(virtOS vos.VOS) int {
	Uptime(virtOS)

	w := tabwriter.NewWriter(virtOS.Stdout(), 0, 8, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "USER\tTTY\tFROM\tLOGIN@\tIDLE\tJCPU\tCPU\tWHAT")
	// TODO: lookup username
	fmt.Fprintf(w, "%s\tpts/0\t%s\t%s\t0.00s\t0.00s\t0.00s\tw\n",
		virtOS.SSHUser(),
		virtOS.SSHRemoteAddr(),
		virtOS.LoginTime().Format("15:04"),
	)

	return 0
}

var _ vos.ProcessFunc = W

func init() {
	addBinCmd("w", W)
	addBinCmd("who", W)
}
