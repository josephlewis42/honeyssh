package commands

import (
	"fmt"
	"os"

	"josephlewis.net/honeyssh/core/vos"
)

// Mkdir implements a POSIX mkdir command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/mkdir.html
func Mkdir(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "mkdir [OPTION...] DIRECTORY...",
		Short: "Create directories if they don't exist.",
	}

	makeParents := cmd.Flags().BoolLong("parents", 'p', "make parents if needed")
	verbose := cmd.Flags().BoolLong("verbose", 'v', "print line for every created directory")

	return cmd.Run(virtOS, func() int {
		directories := cmd.Flags().Args()
		if len(directories) == 0 {
			fmt.Fprintln(virtOS.Stderr(), "mkdir: missing operand")

			cmd.PrintHelp(virtOS.Stdout())
			return 1
		}

		var op func(path string, perm os.FileMode) error
		if *makeParents {
			op = virtOS.MkdirAll
		} else {
			op = virtOS.Mkdir
		}

		anyFailed := false
		for _, dir := range directories {

			err := op(dir, 0777)
			switch {
			case err != nil:
				fmt.Fprintf(virtOS.Stdout(), "mkdir: cannot create directory %q: %s\n", dir, err)
				anyFailed = true

			case *verbose:
				fmt.Fprintf(virtOS.Stdout(), "mkdir: creatred directory: %s\n", dir)
			}
		}

		if anyFailed {
			return 1
		}
		return 0
	})
}

var _ vos.ProcessFunc = Mkdir

func init() {
	addBinCmd("mkdir", Mkdir)
}
