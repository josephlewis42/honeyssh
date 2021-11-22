package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"josephlewis.net/osshit/core/vos"
)

// Touch implements a POSIX touch command.
func Touch(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "touch [OPTION...] FILE...",
		Short: "Update the access and modification times of files to now.",
	}

	// Ignored flags to make the help look more robust. Realistically, access time
	// isn't always recorded by systems for performance reasons.
	cmd.Flags().Bool('a', "only change the access time")
	cmd.Flags().Bool('m', "only change the modification time")

	noCreate := cmd.Flags().BoolLong("no-create", 'c', "don't create files")

	return cmd.Run(virtOS, func() int {
		paths := cmd.Flags().Args()

		now := time.Now()

		var anyFailed bool
		for _, path := range paths {
			err := virtOS.Chtimes(path, now, now)
			switch {
			case errors.Is(err, fs.ErrNotExist) && !*noCreate:
				// ignore error
				fd, err := virtOS.Create(path)
				if err != nil {
					fmt.Fprintf(virtOS.Stderr(), "touch: cannot touch %q: %s\n", path, err)
					anyFailed = true
				}
				fd.Close()
			case errors.Is(err, fs.ErrNotExist) && *noCreate:
				// Not an error.
			case err != nil:
				fmt.Fprintf(virtOS.Stderr(), "touch: setting times of %q: %s\n", path, err)
				anyFailed = true
			}
		}

		if anyFailed {
			return 1
		}
		return 0
	})
}

var _ HoneypotCommandFunc = Touch

func init() {
	addBinCmd("touch", HoneypotCommandFunc(Touch))
}
