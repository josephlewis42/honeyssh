package commands

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"path"

	"josephlewis.net/honeyssh/core/vos"
)

// Unzip implements a basic unzip command.
func Unzip(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "unzip [OPTION...] FILE[.zip]...",
		Short: "Extract files from a zip.",
	}

	return cmd.Run(virtOS, func() int {
		for _, arg := range cmd.Flags().Args() {
			fmt.Fprintf(virtOS.Stdout(), "Archive: %s\n", arg)
			fd, err := virtOS.Open(arg)
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "unzip: %v\n", err)
				return 1
			}
			defer fd.Close()
			stat, err := fd.Stat()
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "unzip: %v\n", err)
				return 1
			}

			reader, err := zip.NewReader(fd, stat.Size())
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "unzip: %v\n", err)
				return 1
			}

			for _, file := range reader.File {
				extractErr := func() error {
					if file.FileInfo().IsDir() {
						fmt.Fprintf(virtOS.Stdout(), "   creating: %s\n", file.Name)
						if err := virtOS.MkdirAll(file.Name, fs.FileMode(0777)); err != nil {
							return err
						}
						return nil
					}

					// Make directories if necessary
					if dir := path.Dir(file.Name); dir != "" {
						if err := virtOS.MkdirAll(dir, fs.FileMode(0777)); err != nil {
							return err
						}
					}

					fmt.Fprintf(virtOS.Stdout(), " extracting: %s\n", file.Name)
					outFd, outErr := virtOS.Create(file.Name)
					if outErr != nil {
						return outErr
					}
					defer outFd.Close()

					zipFd, zipErr := file.Open()
					if zipErr != nil {
						return zipErr
					}
					defer zipFd.Close()

					if _, err := io.Copy(outFd, zipFd); err != nil {
						return err
					}

					return nil
				}()
				if extractErr != nil {
					fmt.Fprintf(virtOS.Stderr(), "unzip: %v\n", extractErr)
				}
			}
		}

		return 0
	})
}

var _ vos.ProcessFunc = Unzip

func init() {
	addBinCmd("unzip", Unzip)
}
