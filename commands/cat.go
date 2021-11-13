package commands

import (
	"flag"
	"fmt"
	"io"

	"josephlewis.net/osshit/core/vos"
)

// Cat implements the UNIX cat command.
func Cat(virtOS vos.VOS) int {
	flags := flag.NewFlagSet("cat", flag.ContinueOnError)
	flags.SetOutput(virtOS.Stderr())
	if err := flags.Parse(virtOS.Args()[1:]); err != nil {
		fmt.Fprintln(virtOS.Stderr(), "Usage: cat [OPTION]... [FILE]...")
		fmt.Fprintln(virtOS.Stderr(), "Concatenate FILE(s) to standard output.")
		return 1
	}

	for _, arg := range virtOS.Args()[1:] {
		fd, err := virtOS.Open(arg)
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "cat: %v\n", err)
			return 1
		}

		io.Copy(virtOS.Stdout(), fd)
		fd.Close()
	}

	return 0
}

var _ HoneypotCommandFunc = Cat

func init() {
	addBinCmd("cat", HoneypotCommandFunc(Cat))
}
