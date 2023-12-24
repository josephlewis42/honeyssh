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
// https://pubs.opengroup.org/onlinepubs/9699919799/utilities/grep.html
func Grep(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "grep [-E|-F] [-iv] PATTERN [FILE]...",
		Short: "Search files for text matching a pattern.",
	}

	invert := cmd.Flags().Bool('v', "Select lines not matching any of the specified patterns.")
	ignoreCase := cmd.Flags().Bool('i', "Perform pattern matching in searches without regard to case.")
	showLineNumbers := cmd.Flags().Bool('n', "Show line numbers.")
	fixedStrings := cmd.Flags().Bool('F', "Treat patterns as strings rather than regular expressions.")
	extendedRegex := cmd.Flags().Bool('E', "Use extended regular expressions.")

	var patternList string
	patternListFlag := cmd.Flags().Flag(&patternList, 'e', "Pattern to be used during the search for input.")

	return cmd.Run(virtOS, func() int {
		args := cmd.Flags().Args()
		// NOTE: Officially, the PATTERN argument supports multiple patterns delimited by newlines.
		// It's a very rare case so we'll ignore it here.
		var pattern string
		switch {
		case patternListFlag.Seen():
			pattern = patternList

		case len(args) == 0:
			cmd.LogProgramError(virtOS, errors.New("missing argument PATTERN"))
			return 1

		default:
			pattern = args[0]
			args = args[1:]
		}

		switch {
		case *extendedRegex || virtOS.Args()[0] == "egrep":
			// Treat the pattern as regex.
		case *fixedStrings:
			pattern = regexp.QuoteMeta(pattern)
		default:
			// TODO: grep should treat the normal mode as "basic" regular expressions
			// rather than extended: https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap09.html#tag_09_03
		}

		if *ignoreCase {
			pattern = "(?i)" + pattern
		}

		regex, err := regexp.Compile(pattern)
		if err != nil {
			cmd.LogProgramError(virtOS, err)
			return 2
		}

		showFileName := len(args) > 1
		return cmd.RunEachFileOrStdin(virtOS, args, func(name string, fd io.Reader) error {
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
	addBinCmd("egrep", Grep)
}
