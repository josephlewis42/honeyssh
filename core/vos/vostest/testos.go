package vostest

import (
	"bytes"
	"io"
	"net"
	"time"

	"github.com/spf13/afero"
	"josephlewis.net/osshit/core/config"
	"josephlewis.net/osshit/core/logger"
	"josephlewis.net/osshit/core/vos"
)

type NopEventRecorder struct{}

func (*NopEventRecorder) Record(event logger.LogType) error {
	return nil
}

type FakeSSHSession struct {
}

func (f *FakeSSHSession) User() string {
	return "$SSHLOGINUSER$"
}

func (f *FakeSSHSession) RemoteAddr() net.Addr {
	return &net.IPNet{IP: net.IPv4(8, 8, 8, 8), Mask: net.IPv4Mask(255, 255, 255, 255)}
}

func (f *FakeSSHSession) Exit(code int) error {
	return nil
}

func (f *FakeSSHSession) Write(b []byte) (int, error) {
	return len(b), nil
}

func SingleProcessResolver(process vos.ProcessFunc) vos.ProcessResolver {
	return func(path string) vos.ProcessFunc {
		return process
	}
}

func NewDeterministicOS(resolver vos.ProcessResolver) vos.VOS {
	timeSource := func() time.Time {
		// Go's reference timestmap with a different value in each position.
		return time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)
	}

	sharedOS := vos.NewSharedOS(afero.NewMemMapFs(), resolver, &config.Configuration{}, timeSource)

	tenantOS := vos.NewTenantOS(sharedOS, &NopEventRecorder{}, &FakeSSHSession{})
	tenantOS.SetPTY(vos.PTY{})

	return tenantOS.InitProc()
}

// Cmd is similar to exec.Cmd.
type Cmd struct {
	// Process function
	Process vos.ProcessFunc
	// Process arguments, the first argument should be the process name.
	Argv []string
	// If Dir is non-empty, the child changes into the directory before
	// creating the process.
	Dir string
	// If Env is non-empty, it gives the environment variables for the
	// new process in the form returned by Environ.
	// If it is nil, the result of Environ will be used.
	Env []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ExitStatus int

	Setup func(vos.VOS) error
}

func Command(process vos.ProcessFunc, name string, arg ...string) *Cmd {
	return &Cmd{
		Process: process,
		Argv:    append([]string{name}, arg...),
	}
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	// stdout, stderr
	buf := &bytes.Buffer{}
	c.Stdout = buf
	c.Stderr = buf

	err := c.Run()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Run starts the comand and waits for it to complete.
func (c *Cmd) Run() error {
	deterministicOS := NewDeterministicOS(SingleProcessResolver(c.Process))
	runner, err := deterministicOS.StartProcess(c.Argv[0], c.Argv, &vos.ProcAttr{
		Dir:   c.Dir,
		Env:   c.Env,
		Files: vos.NewVIOAdapter(io.NopCloser(c.Stdin), writeCloser{c.Stdout}, writeCloser{c.Stderr}),
	})
	if err != nil {
		return err
	}

	if c.Setup != nil {
		if err := c.Setup(runner); err != nil {
			return err
		}
	}

	c.ExitStatus = runner.Run()
	return nil
}

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error {
	return nil
}
