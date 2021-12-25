package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"

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

		dir, err := ioutil.TempDir("", "playground")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)

		playgroundLogger := log.New(cmd.ErrOrStderr(), "[playground] ", 0)
		cfg, err := config.Initialize(dir, playgroundLogger)
		if err != nil {
			return err
		}

		fs, err := vos.NewVFSFromConfig(cfg)
		if err != nil {
			return err
		}

		playgroundLogger.Printf("Logging to: file://%s\n", dir)
		playgroundLogger.Println(strings.Repeat("=", 80))

		sharedOS := vos.NewSharedOS(fs, commands.BuiltinProcessResolver, cfg, time.Now)
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

		initProc := tenantOS.LoginProc()

		runner, err := initProc.StartProcess("/bin/sh", []string{}, &vos.ProcAttr{
			Dir:   "/",
			Env:   initProc.Environ(),
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
