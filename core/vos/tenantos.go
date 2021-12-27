package vos

import (
	"io"
	"net"
	"time"

	"josephlewis.net/honeyssh/core/logger"
)

type TenantOS struct {
	*SharedOS
	// fs contains a tenant's view of the shared OS.
	fs VFS
	// eventRecorder logs events.
	eventRecorder EventRecorder
	// Connected terminal information.
	pty PTY
	// loginTime is the time the user logged in
	loginTime time.Time

	session SSHSession
}

type EventRecorder interface {
	Record(event logger.LogType) error
	SessionID() string
}

type SSHSession interface {
	User() string
	RemoteAddr() net.Addr
	Exit(code int) error
	Write([]byte) (int, error)
}

func NewTenantOS(sharedOS *SharedOS, eventRecorder EventRecorder, session SSHSession) *TenantOS {
	mountFS := NewMountFS(sharedOS.ReadOnlyFs())
	if err := mountFS.Mount("/proc", NewProcFS(sharedOS)); err != nil {
		panic(err)
	}

	ufs := NewMemCopyOnWriteFs(mountFS, sharedOS.timeSource)

	return &TenantOS{
		SharedOS:      sharedOS,
		fs:            ufs,
		eventRecorder: eventRecorder,
		loginTime:     sharedOS.timeSource(),
		session:       session,
	}
}

func (t *TenantOS) SetPTY(pty PTY) {
	t.eventRecorder.Record(&logger.LogEntry_TerminalUpdate{
		TerminalUpdate: &logger.TerminalUpdate{
			Width:  int32(pty.Width),
			Height: int32(pty.Height),
			Term:   pty.Term,
			IsPty:  pty.IsPTY,
		},
	})

	t.pty = pty
}

func (t *TenantOS) GetPTY() PTY {
	return t.pty
}

func (t *TenantOS) LoginProc() *TenantProcOS {
	env := NewMapEnvFromEnvList(t.loginEnv())
	usr, _ := t.SharedOS.GetUser(t.SSHUser())
	return &TenantProcOS{
		TenantOS:       t,
		VFS:            t.fs,
		VIO:            NewNullIO(),
		VEnv:           env,
		ExecutablePath: "/sbin/sshd",
		ProcArgs:       []string{"/sbin/sshd"},
		PID:            0,
		UID:            usr.UID,
		Dir:            env.Getenv("PWD"),
		Exec: func(_ VOS) int {
			return 0
		},
	}
}

func (t *TenantOS) LoginTime() time.Time {
	return t.loginTime
}

// SSHUser returns the username used when establishing the SSH connection.
func (t *TenantOS) SSHUser() string {
	return t.session.User()
}

// SSHRemoteAddr returns the net.Addr of the client side of the connection.
func (t *TenantOS) SSHRemoteAddr() net.Addr {
	return t.session.RemoteAddr()
}

// SSHStdout is a direct connection to the SSH stdout stream.
// Useful for broadcasting messages.
func (t *TenantOS) SSHStdout() io.Writer {
	return t.session
}

// SSHExit hangs up the incoming SSH connection.
func (t *TenantOS) SSHExit(code int) error {
	return t.session.Exit(code)
}

// LogCreds records credentials that the attacker used.
func (t *TenantOS) LogCreds(creds *logger.Credentials) {
	t.eventRecorder.Record(&logger.LogEntry_UsedCredentials{
		UsedCredentials: creds,
	})
}

func (t *TenantOS) loginEnv() []string {
	mapEnv := NewMapEnv()

	mapEnv.Setenv("SHELL", t.SharedOS.config.OS.DefaultShell)
	mapEnv.Setenv("PATH", t.SharedOS.config.OS.DefaultPath)
	mapEnv.Setenv("PWD", "/")
	mapEnv.Setenv("HOME", "/")

	username := t.session.User()
	mapEnv.Setenv("USER", username)
	mapEnv.Setenv("LOGNAME", username)

	if usr, ok := t.SharedOS.GetUser(username); ok {
		if usr.Shell != "" {
			mapEnv.Setenv("SHELL", usr.Shell)
		}
		if usr.Home != "" {
			mapEnv.Setenv("PWD", usr.Home)
			mapEnv.Setenv("HOME", usr.Home)
		}
	}

	if term := t.GetPTY().Term; term != "" {
		mapEnv.Setenv("TERM", term)
	}

	return mapEnv.Environ()
}
