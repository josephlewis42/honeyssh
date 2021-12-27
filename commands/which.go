package commands

import (
	"fmt"

	"josephlewis.net/honeyssh/core/vos"
)

// Which implements the UNIX which command.
func Which(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "which [COMMAND...]",
		Short: "Locate a command.",
		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.RunEachArg(virtOS, func(arg string) error {
		res, err := vos.LookPath(virtOS, arg)
		if err != nil {
			return err
		}
		fmt.Fprintln(virtOS.Stdout(), res)
		return nil
	})
}

var _ vos.ProcessFunc = Which

func init() {
	addBinCmd("which", Which)
}
