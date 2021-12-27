package commands

import (
	"fmt"

	"josephlewis.net/honeyssh/core/vos"
)

// Php implements a fake PHP interpreter.
func Php(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "php [options] [-f] <file> [--] [args...]",
		Short: "PHP Command Line Interface.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtOS, func() int {
		w := virtOS.Stdout()
		fmt.Fprintln(w, `PHP:  Error parsing php.ini on line 424`)
		return 1
	})
}

var _ vos.ProcessFunc = Php

func init() {
	addBinCmd("php", Php)
}
