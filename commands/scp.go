package commands

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"strconv"

	goscp "github.com/bramvdbogaerde/go-scp"
	"github.com/josephlewis42/honeyssh/core/vos"
)

//
func scpUpload(vos vos.VOS, path string) (err error) {
	// Start upload in VOS
	uploadFd, err := vos.DownloadPath(fmt.Sprintf("scp_upload://%s", path))
	if err != nil {
		fmt.Fprintln(vos.Stderr(), "Error", err)
		return err
	}
	defer uploadFd.Close()
	tarWriter := tar.NewWriter(uploadFd)
	defer tarWriter.Close()

	// Start the session by sending an ACK
	goscp.Ack(vos.Stdout())

	for {
		resp, err := goscp.ParseResponse(vos.Stdin())
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}

		fmt.Printf("Receiving message: %d %q\n", resp.Type, resp.Message)

		switch resp.Type {
		// OK or non-fatal error:
		case 0x00, 0x01:
			goscp.Ack(vos.Stdout())
			continue

		case 0x02:
			return fmt.Errorf("fatal error")
		case 'E': // exit
			return nil

		case 'C': // File transfer
			fileInfo, err := resp.ParseFileInfos()
			if err != nil {
				return err
			}
			mode, err := strconv.ParseInt(fileInfo.Permissions, 8, 64)
			if err != nil {
				return fmt.Errorf("bad mode %q", fileInfo.Permissions)
			}

			if err := tarWriter.WriteHeader(&tar.Header{
				Name: fileInfo.Filename,
				Mode: mode,
				Size: fileInfo.Size,
			}); err != nil {
				return err
			}
			goscp.Ack(vos.Stdout())

			if _, err := io.CopyN(tarWriter, vos.Stdin(), fileInfo.Size); err != nil {
				if err != io.EOF {
					return err
				}
			}
			goscp.Ack(vos.Stdout())

		case 'T', 'D': // Set timestamps for next file; directory
			goscp.Ack(vos.Stdout())
		default:
			return errors.New("unknown directive")
		}
	}
}

// Scp implements an SCP command that only uploads.
func Scp(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "scp -t TOFILE",
		Short: "Secure copy.",

		// Never bail, even if args are bad.
		NeverBail: true,
	}

	to := cmd.Flags().String('t', "", "Start scp in upload mode")
	_ = cmd.Flags().Bool('v', "Start scp in verbose mode")
	_ = cmd.Flags().Bool('r', "Start scp in recursive mode")

	return cmd.RunE(virtOS, func() error {
		switch {
		case *to != "":
			err := scpUpload(virtOS, *to)
			if err != nil {
				fmt.Fprintln(virtOS.Stderr(), err.Error())
				fmt.Println(err.Error())
				return err
			}
			return nil
		default:
			return errors.New("couldn't connect")
		}
	})
}

var _ vos.ProcessFunc = Scp

func init() {
	addBinCmd("scp", Scp)
}
