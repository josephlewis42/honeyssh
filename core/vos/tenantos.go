package vos

import (
	"io"
	"net"
	"time"

	"josephlewis.net/osshit/core/logger"
)

type TenantOS struct {
	sharedOS *SharedOS
	// fs contains a tenant's view of the shared OS.
	fs VFS
	// eventRecorder logs events.
	eventRecorder EventRecorder
	// Connected terminal information.
	pty PTY
	// loginTime is the time the user logged in
	loginTime time.Time
	// Username the user logged in as.
	user string
	// Remote address of the connected user.
	remoteAddr net.Addr

	sshStdout io.Writer

	sshExit func(int) error
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
	ufs := NewMemCopyOnWriteFs(sharedOS.ReadOnlyFs(), sharedOS.timeSource)

	return &TenantOS{
		sharedOS:      sharedOS,
		fs:            ufs,
		eventRecorder: eventRecorder,
		loginTime:     sharedOS.timeSource(),
		user:          session.User(),
		remoteAddr:    session.RemoteAddr(),
		sshExit:       session.Exit,
		sshStdout:     session,
	}
}

// Hostname implements VOS.Hostname.
func (t *TenantOS) Hostname() string {
	return t.sharedOS.Hostname()
}

// Uname implements VOS.Uname.
func (t *TenantOS) Uname() Utsname {
	return t.sharedOS.Uname()
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
	env := NewMapEnvFromEnvList(t.sharedOS.LoginEnv(t.user))
	usr, _ := t.sharedOS.GetUser(t.user)
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

// TODO: add a LoginProc() to set the variables for the login process
// it should set uid, gid, $SHELL, $PATH, $LOGNAME

func (t *TenantOS) BootTime() time.Time {
	return t.sharedOS.bootTime
}

func (t *TenantOS) LoginTime() time.Time {
	return t.loginTime
}

// SSHUser returns the username used when establishing the SSH connection.
func (t *TenantOS) SSHUser() string {
	return t.user
}

// SSHRemoteAddr returns the net.Addr of the client side of the connection.
func (t *TenantOS) SSHRemoteAddr() net.Addr {
	if t.remoteAddr == nil {
		return &net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 255, 255, 255)}
	}
	return t.remoteAddr
}

// SSHStdout is a direct connection to the SSH stdout stream.
// Useful for broadcasting messages.
func (t *TenantOS) SSHStdout() io.Writer {
	return t.sshStdout
}

// SSHExit hangs up the incoming SSH connection.
func (t *TenantOS) SSHExit(code int) error {
	return t.sshExit(code)
}

// LogCreds records credentials that the attacker used.
func (t *TenantOS) LogCreds(creds *logger.Credentials) {
	t.eventRecorder.Record(&logger.LogEntry_UsedCredentials{
		UsedCredentials: creds,
	})
}

func (t *TenantOS) Now() time.Time {
	return t.sharedOS.timeSource()
}
