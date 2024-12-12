package cmd

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/anmitsu/go-shlex"
	"github.com/gliderlabs/ssh"
	"github.com/josephlewis42/honeyssh/core"
	"github.com/josephlewis42/honeyssh/core/config"
	"github.com/josephlewis42/honeyssh/core/vos"
	"github.com/spf13/cobra"
)

type playgroundSession struct {
	user       string
	out        io.Writer
	in         io.Reader
	rawCommand string
	subsystem  string
	environ    []string
	pty        ssh.Pty

	exitCalled bool
	exitCode   int
}

var _ core.SessionInfo = (*playgroundSession)(nil)

func (p *playgroundSession) User() string {
	return p.user
}

func (p *playgroundSession) RemoteAddr() net.Addr {
	return &net.IPNet{IP: net.IPv4(8, 8, 8, 8), Mask: net.IPv4Mask(255, 255, 255, 255)}
}

func (p *playgroundSession) Exit(code int) error {
	p.exitCalled = true
	p.exitCode = code
	return nil
}

func (p *playgroundSession) Write(b []byte) (int, error) {
	return p.out.Write(b)
}

func (p *playgroundSession) Command() []string {
	// Ignore the error, it doesn't matter for the playground.
	cmd, _ := shlex.Split(p.rawCommand, true)
	return cmd
}

func (p *playgroundSession) RawCommand() string {
	return p.rawCommand
}
func (p *playgroundSession) Subsystem() string {
	return p.subsystem
}

func (p *playgroundSession) Context() context.Context {
	return context.Background()
}

func (p *playgroundSession) Environ() []string {
	return p.environ
}

func (p *playgroundSession) Pty() (ssh.Pty, <-chan ssh.Window, bool) {
	output := make(<-chan ssh.Window)
	return p.pty, output, true
}

func (p *playgroundSession) Read(b []byte) (int, error) {
	return p.in.Read(b)
}

func (p *playgroundSession) Close() error {
	// Close does nothing in playground
	return nil
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

		dir, err := os.MkdirTemp("", "playground")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dir)

		playgroundLogger := log.New(cmd.ErrOrStderr(), "[playground] ", 0)
		cfg, err := config.Initialize(dir, playgroundLogger)
		if err != nil {
			return err
		}

		// Add a honeypot to the hostname to help differentiate the fake shell from
		// a real one -- it's surprisingly convincing.
		cfg.Uname.Nodename = "playground🍯"

		playgroundLogger.Printf("Logging to: file://%s\n", dir)
		playgroundLogger.Printf("See logs with: tail -f %s\n", filepath.Join(dir, config.AppLogName))
		playgroundLogger.Println(strings.Repeat("=", 80))

		honeypot, err := core.NewHoneypot(cfg, io.Discard)
		if err != nil {
			return err
		}

		session := &playgroundSession{
			out:        cmd.OutOrStdout(),
			in:         cmd.InOrStdin(),
			user:       "root",
			rawCommand: "/bin/sh",
			subsystem:  "",
			environ:    []string{},
			pty: ssh.Pty{
				Term: "playground",
				Window: ssh.Window{
					Width:  80,
					Height: 40,
				},
			},
		}

		if err := honeypot.HandleConnection(session); err != nil {
			return err
		}

		playgroundLogger.Println(strings.Repeat("=", 80))
		if session.exitCalled {
			playgroundLogger.Printf("Session ended, exit code: %d\n", session.exitCode)
		} else {
			playgroundLogger.Println("Session ended, exit not called")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(playgroundCmd)
}
