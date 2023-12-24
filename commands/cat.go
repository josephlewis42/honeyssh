package commands

import (
	"io"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Cat implements the POSIX cat command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/
func Cat(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "cat [OPTION]... [FILE]...",
		Short: "Concatenate FILE(s) to standard output.",
	}

	return cmd.RunEachArg(virtOS, func(path string) error {
		fd, err := virtOS.Open(path)
		if err != nil {
			return err
		}
		defer fd.Close()

		_, err = io.Copy(virtOS.Stdout(), fd)
		return err
	})
}

var _ vos.ProcessFunc = Cat

func init() {
	mustAddBinCmd("cat", Cat)
}
