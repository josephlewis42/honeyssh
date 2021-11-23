package commands

import (
	"fmt"

	"github.com/abiosoft/readline"
	"josephlewis.net/osshit/core/logger"
	"josephlewis.net/osshit/core/vos"
)

// Passwd implements a fake passwd command.
func Passwd(virtualOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "passwd [OPTION] [LOGIN]",
		Short: "Change user password.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	return cmd.Run(virtualOS, func() int {
		cfg := &readline.Config{
			Stdin:  readline.NewCancelableStdin(virtualOS.Stdin()),
			Stdout: virtualOS.Stdout(),
			Stderr: virtualOS.Stderr(),
			FuncGetWidth: func() int {
				return virtualOS.GetPTY().Width
			},
			FuncIsTerminal: func() bool {
				return virtualOS.GetPTY().IsPTY
			},
		}
		if err := cfg.Init(); err != nil {
			return 1
		}
		readline, err := readline.NewEx(cfg)
		if err != nil {
			return 1
		}
		defer readline.Close()

		login := virtualOS.SSHUser()
		if args := cmd.Flags().Args(); len(args) > 0 {
			login = args[0]
		}

		newPass1, err1 := readline.ReadPassword("Enter new UNIX password: ")
		if err1 != nil {
			return 1
		}
		// Regardless of whether they match, record the creds.
		virtualOS.LogCreds(&logger.Credentials{
			Username: login,
			Password: string(newPass1),
		})
		newPass2, err2 := readline.ReadPassword("Retype new UNIX password: ")
		if err2 != nil {
			return 1
		}

		if string(newPass1) != string(newPass2) {
			fmt.Fprintln(virtualOS.Stdout(), "Sorry, passwords don't match.")
			fmt.Fprintln(virtualOS.Stdout(), "passwd: password unchanged")
			return 0
		}
		fmt.Fprintln(virtualOS.Stdout(), "passwd: password updated successfully")

		return 0
	})
}

var _ HoneypotCommandFunc = Passwd

func init() {
	addBinCmd("passwd", HoneypotCommandFunc(Passwd))
}
