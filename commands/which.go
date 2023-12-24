package commands

import (
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
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
	mustAddBinCmd("which", Which)
}
