package commands

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"josephlewis.net/osshit/core/vos"
)

// Rmdir implements a POSIX rmdir command.
func Rmdir(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "rmdir [OPTION...] DIRECTORY...",
		Short: "Remove empty directories.",
	}

	parents := cmd.Flags().BoolLong("parents", 'p', "make parents if needed")
	verbose := cmd.Flags().BoolLong("verbose", 'v', "print line for every deleted directory")

	return cmd.Run(virtOS, func() int {
		directories := cmd.Flags().Args()
		if len(directories) == 0 {
			fmt.Fprintln(virtOS.Stderr(), "mkdir: missing operand")

			cmd.PrintHelp(virtOS.Stdout())
			return 1
		}

		anyFailed := false
		for _, dir := range directories {
			steps := []string{}
			if *parents {
				var built []string
				for _, p := range strings.Split(dir, "/") {
					built = append(built, p)
					steps = append(steps, path.Join(built...))
				}
				// Sort longest to shortest for depth.
				sort.Slice(steps, func(i, j int) bool {
					return len(steps[i]) > len(steps[j])
				})
			} else {
				steps = append(steps, dir)
			}

			for _, dir := range steps {
				file, err := virtOS.Open(dir)
				if err != nil {
					fmt.Fprintf(virtOS.Stderr(), "rmdir: cannot read directory %q: %s\n", dir, err)
					anyFailed = true
					break
				}

				contents, err := file.Readdir(-1)
				file.Close()
				if err != nil {
					fmt.Fprintf(virtOS.Stderr(), "rmdir: cannot read directory %q: %s\n", dir, err)
					anyFailed = true
					break
				}

				if len(contents) > 0 {
					fmt.Fprintf(virtOS.Stderr(), "rmdir: directory not empty %q\n", dir)
					break
				}

				// Remove
				err = virtOS.Remove(dir)
				switch {
				case err != nil:
					fmt.Fprintf(virtOS.Stderr(), "rmdir: cannot remove directory %q: %s\n", dir, err)
					anyFailed = true
					break

				case *verbose:
					fmt.Fprintf(virtOS.Stdout(), "rmdir: removed directory: %s\n", dir)
				}
			}
		}

		if anyFailed {
			return 1
		}
		return 0
	})
}

var _ HoneypotCommandFunc = Rmdir

func init() {
	addBinCmd("rmdir", HoneypotCommandFunc(Rmdir))
}
