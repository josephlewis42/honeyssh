package vostest

import (
	"bytes"
	"io"
	"net"
	"time"

	"josephlewis.net/osshit/core/config"
	"josephlewis.net/osshit/core/logger"
	"josephlewis.net/osshit/core/vos"
	"josephlewis.net/osshit/third_party/memmapfs"
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

func NewDeterministicOS(resolver vos.ProcessResolver) vos.VOS {
	timeSource := func() time.Time {
		// Go's reference timestmap with a different value in each position.
		return time.Date(2006, 1, 2, 3, 4, 5, 0, time.UTC)
	}

	sharedOS := vos.NewSharedOS(memmapfs.NewMemMapFs(timeSource), resolver, &config.Configuration{}, timeSource)

	tenantOS := vos.NewTenantOS(sharedOS, &NopEventRecorder{}, &FakeSSHSession{})
	tenantOS.SetPTY(vos.PTY{})

	return tenantOS.LoginProc()
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

	// VOS will be initialized after Command is called.
	VOS vos.VOS

	Setup func(vos.VOS) error
}

func (c *Cmd) processResolver(path string) vos.ProcessFunc {
	return c.Process
}

func Command(process vos.ProcessFunc, name string, arg ...string) *Cmd {
	cmd := &Cmd{
		Process: process,
		Argv:    append([]string{name}, arg...),
	}
	cmd.VOS = NewDeterministicOS(cmd.processResolver)

	return cmd
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
	runner, err := c.VOS.StartProcess(c.Argv[0], c.Argv, &vos.ProcAttr{
		Dir:   c.Dir,
		Env:   c.Env,
		Files: vos.NewVIOAdapter(io.NopCloser(c.Stdin), newWriteCloser(c.Stdout), newWriteCloser(c.Stderr)),
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

func newWriteCloser(w io.Writer) io.WriteCloser {
	if w == nil {
		return nil
	}
	return &writeCloser{w}
}

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error {
	return nil
}
