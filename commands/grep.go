package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Grep implements the POSIX grep command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/
func Grep(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "grep [-iv] PATTERN [FILE]...",
		Short: "Search files for text matching a pattern.",
	}

	invert := cmd.Flags().Bool('v', "Select lines not matching any of the specified patterns.")
	ignoreCase := cmd.Flags().Bool('i', "Perform pattern matching in searches without regard to case.")
	showLineNumbers := cmd.Flags().Bool('n', "Show line numbers.")

	return cmd.Run(virtOS, func() int {
		args := cmd.Flags().Args()
		if len(args) == 0 {
			cmd.LogProgramError(virtOS, errors.New("missing argument PATTERN"))
			return 1
		}

		// NOTE: Officially, the PATTERN argument supports multiple patterns delimited by newlines.
		// It's a very rare case so we'll ignore it here.
		pattern := args[0]
		if *ignoreCase {
			pattern = "(?i)" + pattern
		}
		regex, err := regexp.Compile(pattern)
		if err != nil {
			cmd.LogProgramError(virtOS, err)
			return 2
		}

		files := args[1:]
		showFileName := len(files) > 1
		return cmd.RunEachFileOrStdin(virtOS, files, func(name string, fd io.Reader) error {
			w := virtOS.Stdout()

			scanner := bufio.NewScanner(fd)
			lineNo := 1
			for scanner.Scan() {
				line := scanner.Bytes()
				lineMatches := regex.Match(line)

				// Write output
				if (lineMatches && !*invert) || (!lineMatches && *invert) {
					if showFileName {
						fmt.Fprintf(w, "%s:", name)
					}

					if *showLineNumbers {
						fmt.Fprintf(w, "%d:", lineNo)
					}

					fmt.Fprintf(w, "%s\n", line)
				}
				lineNo++
			}

			return nil
		})
	})
}

var _ vos.ProcessFunc = Grep

func init() {
	addBinCmd("grep", Grep)
}
