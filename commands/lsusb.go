package commands

import (
	"fmt"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
)

var (
	lsusbText = strings.TrimSpace(`
Bus 001 Device 001: ID 1d6b:0001 Linux Foundation 1.1 root hub
	`)
)

// Lsusb implements the lsusb command.
func Lsusb(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "lspci [OPTION...]",
		Short: "List USB devices.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		fmt.Fprintln(virtOS.Stdout(), lsusbText)
		return 0
	})
}

var _ vos.ProcessFunc = Lsusb

func init() {
	addBinCmd("lsusb", Lsusb)
}
