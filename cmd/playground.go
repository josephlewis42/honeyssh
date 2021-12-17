package cmd

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"josephlewis.net/osshit/commands"
	"josephlewis.net/osshit/core/config"
	"josephlewis.net/osshit/core/vos"
	"josephlewis.net/osshit/core/vos/vostest"
)

type playgroundSession struct {
	user string
	out  io.Writer
}

func (p *playgroundSession) User() string {
	return p.user
}

func (p *playgroundSession) RemoteAddr() net.Addr {
	return &net.IPNet{IP: net.IPv4(8, 8, 8, 8), Mask: net.IPv4Mask(255, 255, 255, 255)}
}

func (p *playgroundSession) Exit(code int) error {
	os.Exit(code)
	return nil
}

func (p *playgroundSession) Write(b []byte) (int, error) {
	return p.out.Write(b)
}

type SSHSession interface {
	User() string
	RemoteAddr() net.Addr
	Exit(code int) error
	Write([]byte) (int, error)
}

type osVIO struct {
}

func (c *osVIO) Stderr() io.WriteCloser {
	return os.Stderr
}

func (c *osVIO) Stdout() io.WriteCloser {
	return os.Stdout
}

func (c *osVIO) Stdin() io.ReadCloser {
	return os.Stdin
}

var _ vos.VIO = (*osVIO)(nil)

// playgroundCmd runs the honeypot shell over the local OS for testing
var playgroundCmd = &cobra.Command{
	Use:   "playground",
	Short: "Run the honeypot shell without staring a server or logging.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		fs := vos.NewMemCopyOnWriteFs(afero.NewReadOnlyFs(afero.NewOsFs()), time.Now)

		sharedOS := vos.NewSharedOS(fs, commands.BuiltinProcessResolver, &config.Configuration{}, time.Now)

		tenantOS := vos.NewTenantOS(sharedOS, &vostest.NopEventRecorder{}, &playgroundSession{
			out:  cmd.OutOrStdout(),
			user: "root",
		})
		// TODO: Connect to the real PTY
		tenantOS.SetPTY(vos.PTY{
			Width:  80,
			Height: 40,
			Term:   "playground",
			IsPTY:  true,
		})

		initProc := tenantOS.InitProc()

		runner, err := initProc.StartProcess("/bin/sh", []string{}, &vos.ProcAttr{
			Dir:   "/",
			Env:   []string{},
			Files: &osVIO{},
		})
		if err != nil {
			return err
		}

		exitCode := runner.Run()
		fmt.Fprintf(cmd.OutOrStdout(), "Exit code: %d\n", exitCode)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(playgroundCmd)
}
