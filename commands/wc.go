package commands

import (
	"fmt"
	"io"
	"unicode"

	"github.com/josephlewis42/honeyssh/core/vos"
)

type wcCount struct {
	bytes int
	lines int
	chars int
	words int
	name  string

	inSpace bool
}

func (w *wcCount) Write(data []byte) (int, error) {
	for _, c := range data {
		isFirstByte := w.bytes == 0
		w.bytes++

		// Assume UTF-8 characters. Bytes following the leading byte always
		// have MSB of 0b10 indicating they're part of a previous character.
		if c < 0b10000000 || c > 0b10111111 {
			w.chars++
		}

		if c == '\n' {
			w.lines++
		}

		if unicode.IsSpace(rune(c)) {
			w.inSpace = true
		} else {
			if w.inSpace || isFirstByte {
				w.words++
			}
			w.inSpace = false
		}
	}

	return len(data), nil
}

func NewWcCount(name string, fd io.Reader) (*wcCount, error) {
	var out wcCount
	out.name = name

	if _, err := io.Copy(&out, fd); err != nil {
		return nil, err
	}

	return &out, nil
}

func (w *wcCount) Increment(other *wcCount) {
	w.bytes += other.bytes
	w.chars += other.chars
	w.lines += other.lines
	w.words += other.words
}

// Wc implements the POSIX command by the same name.
// https://pubs.opengroup.org/onlinepubs/009695399/utilities/wc.html
func Wc(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "wc [-c|-m] [-lw] [FILE...]",
		Short: "Write the number of newlines, words, and bytes contained in each input file to the standard output.",
	}

	opts := cmd.Flags()
	writeLines := opts.BoolLong("l", 'l', "write the number of newlines in each file")
	writeWords := opts.BoolLong("w", 'w', "write the number of words in each file")
	writeBytes := opts.BoolLong("c", 'c', "write the number of bytes in each file")
	writeChars := opts.BoolLong("m", 'm', "write the number of characters in each file")

	return cmd.RunE(virtOS, func() error {
		args := opts.Args()

		anyPicked := *writeLines || *writeWords || *writeBytes || *writeChars
		nonePicked := !anyPicked

		var cols []func(*wcCount) string

		if *writeLines || nonePicked {
			cols = append(cols, func(w *wcCount) string {
				return fmt.Sprint(w.lines)
			})
		}
		if *writeWords || nonePicked {
			cols = append(cols, func(w *wcCount) string {
				return fmt.Sprint(w.words)
			})
		}
		if *writeBytes || nonePicked {
			cols = append(cols, func(w *wcCount) string {
				return fmt.Sprint(w.bytes)
			})
		}
		if *writeChars {
			cols = append(cols, func(w *wcCount) string {
				return fmt.Sprint(w.chars)
			})
		}

		displayCount := func(count *wcCount) {
			for i, col := range cols {
				if i != 0 {
					fmt.Fprint(virtOS.Stdout(), " ")
				}
				fmt.Fprint(virtOS.Stdout(), col(count))
			}
			fmt.Fprintln(virtOS.Stdout())
		}

		writeCounts := func(countsList ...*wcCount) {
			total := &wcCount{name: "total"}

			for _, count := range countsList {
				total.Increment(count)
				displayCount(count)
			}

			if len(countsList) > 1 {
				displayCount(total)
			}
		}

		if len(args) == 0 {
			count, err := NewWcCount("", virtOS.Stdin())
			if err != nil {
				return err
			}
			writeCounts(count)
			return nil
		}

		cols = append(cols, func(w *wcCount) string {
			return w.name
		})

		var counts []*wcCount
		for _, path := range args {
			fd, err := virtOS.Open(path)
			if err != nil {
				return err
			}
			defer fd.Close()

			count, err := NewWcCount(path, fd)
			if err != nil {
				return err
			}

			counts = append(counts, count)
		}

		writeCounts(counts...)

		return nil
	})
}

var _ vos.ProcessFunc = Wc

func init() {
	mustAddBinCmd("wc", Wc)
}
