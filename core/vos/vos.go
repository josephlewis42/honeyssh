package vos

import (
	"net"
	"time"
)

type VNetwork interface {
	Hostname() (string, error)
}

type PTY struct {
	Width  int
	Height int
	Term   string
	IsPTY  bool
}

// VOS provides a virtual OS interface.
type VOS interface {
	VNetwork
	VEnv
	VIO
	VProc
	VFS
	Honeypot
}

type Honeypot interface {
	// BootTime provides a fake boot itme.
	BootTime() time.Time
	// LoginTime provides the time the session started.
	LoginTime() time.Time
	// SSHUser returns the username used when establishing the SSH connection.
	SSHUser() string
	// SSHRemoteAddr returns the net.Addr of the client side of the connection.
	SSHRemoteAddr() net.Addr

	SetPTY(PTY)
	GetPTY() PTY

	StartProcess(name string, argv []string, attr *ProcAttr) (VOS, error)
}
