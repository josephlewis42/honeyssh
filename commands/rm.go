package commands

import (
	"errors"
	"fmt"
	"io/fs"

	"josephlewis.net/osshit/core/vos"
)

// Rm implements a POSIX rm command.
func Rm(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "rm [OPTION...] FILE...",
		Short: "Remove files or directories.",
	}

	recursive := cmd.Flags().BoolLong("recursive", 'r', "remove directories and their contents recursively")
	force := cmd.Flags().BoolLong("force", 'f', "ignore missing files and arguments, never prompt")

	return cmd.Run(virtOS, func() int {
		anyFailed := false
		for _, file := range cmd.Flags().Args() {
			stat, statErr := virtOS.Stat(file)
			switch {
			case errors.Is(statErr, fs.ErrNotExist):
				if !*force {
					fmt.Fprintf(virtOS.Stderr(), "rm: can't remove %q: no such file or directory\n", file)
					anyFailed = true
				}
			case statErr != nil:
				fmt.Fprintf(virtOS.Stderr(), "rm: can't stat %q: %v\n", file, statErr)
				anyFailed = true
			case stat.Mode().IsDir():
				if !*recursive {
					fmt.Fprintf(virtOS.Stderr(), "rm: can't remove %q: is a directory\n", file)
					anyFailed = true
					continue
				}
				fallthrough
			default:
				// regular file, remove
				if err := virtOS.Remove(file); err != nil {
					fmt.Fprintf(virtOS.Stderr(), "rm: can't remove %q: %v\n", file, err)
					anyFailed = true
				}
			}
		}

		if anyFailed {
			return 1
		}
		return 0
	})
}

var _ HoneypotCommandFunc = Rm

func init() {
	addBinCmd("rm", Rm)
}
