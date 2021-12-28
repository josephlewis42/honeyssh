package commands

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/josephlewis42/honeyssh/core/vos"
)

// Tar implements a basic tar command.
func Tar(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "tar [OPTION...] FILE...",
		Short: "Modify Tape ARchives.",

		NeverBail: true,
	}

	extract := cmd.Flags().Bool('x', "Extract files")
	verbose := cmd.Flags().Bool('v', "Verbose mode")
	unzip := cmd.Flags().Bool('z', "Unzip archive")
	archive := cmd.Flags().String('f', "The archive to use")

	return cmd.Run(virtOS, func() int {
		if archive == nil || *archive == "" {
			fmt.Fprintf(virtOS.Stderr(), "tar: no archive supplied, use -f\n")
			return 1
		}

		archiveFd, err := virtOS.Open(*archive)
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "tar: couldn't open archive %q: %v\n", *archive, err)
			return 1
		}
		defer archiveFd.Close()
		var tarFd io.Reader = archiveFd

		if *unzip {
			gzFd, err := gzip.NewReader(archiveFd)
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "tar: couldn't unzip: %v\n", err)
				return 1
			}
			tarFd = gzFd
		}

		tarReader := tar.NewReader(tarFd)

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "tar: couldn't read file: %v\n", err)
				return 1
			}

			name := strings.TrimPrefix(hdr.FileInfo().Name(), "/")
			if *verbose {
				fmt.Fprintln(virtOS.Stdout(), name)
			}

			switch {
			case !*extract:
				continue
			case hdr.FileInfo().IsDir():
				if err := virtOS.MkdirAll(name, fs.FileMode(hdr.FileInfo().Mode())); err != nil {
					fmt.Fprintf(virtOS.Stderr(), "tar: couldn't extract %q: %v\n", name, err)
					return 1
				}
			default:
				if dir := path.Dir(name); dir != "" {
					if err := virtOS.MkdirAll(dir, fs.FileMode(0777)); err != nil {
						fmt.Fprintf(virtOS.Stderr(), "tar: couldn't extract %q: %v\n", name, err)
						return 1
					}
				}
				outFd, outErr := virtOS.Create(name)
				if outErr != nil {
					fmt.Fprintf(virtOS.Stderr(), "tar: couldn't extract %q: %v\n", name, err)
					return 1
				}
				defer outFd.Close()

				if _, err := io.Copy(outFd, tarReader); err != nil {
					fmt.Fprintf(virtOS.Stderr(), "tar: couldn't extract %q: %v", name, err)
					return 1
				}
			}
		}

		return 0
	})
}

var _ vos.ProcessFunc = Tar

func init() {
	addBinCmd("tar", Tar)
}
