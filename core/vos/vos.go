package vos

import (
	"io"
	"net"
	"time"

	"josephlewis.net/osshit/core/logger"
)

// Utsname mimics POSIX sys/utsname.h
// https://pubs.opengroup.org/onlinepubs/7908799/xsh/sysutsname.h.html
type Utsname struct {
	Sysname    string // OS name e.g. "Linux".
	Nodename   string // Hostname of the machine on one of its networks.
	Release    string // OS release e.g. "4.15.0-147-generic"
	Version    string // OS version e.g. "#151-Ubuntu SMP Fri Jun 18 19:21:19 UTC 2021"
	Machine    string // Machnine name e.g. "x86_64"
	Domainname string // NIS or YP domain name
}

type VKernel interface {
	Hostname() (string, error)
	// Uname mimics the uname syscall.
	Uname() (Utsname, error)
}

type PTY struct {
	Width  int
	Height int
	Term   string
	IsPTY  bool
}

// VOS provides a virtual OS interface.
type VOS interface {
	VKernel
	VEnv
	VIO
	VProc
	VFS
	Honeypot
}

// Honeypot contains non-OS utilities related to running the honeypot.
type Honeypot interface {
	// BootTime provides a fake boot itme.
	BootTime() time.Time
	// LoginTime provides the time the session started.
	LoginTime() time.Time
	// SSHUser returns the username used when establishing the SSH connection.
	SSHUser() string
	// SSHRemoteAddr returns the net.Addr of the client side of the connection.
	SSHRemoteAddr() net.Addr
	// Write to the attahed SSH session's output.
	SSHStdout() io.Writer
	// Exit the attached SSH session.
	SSHExit(int) error

	SetPTY(PTY)
	GetPTY() PTY

	StartProcess(name string, argv []string, attr *ProcAttr) (VOS, error)

	// Log an invalid command invocation, it may indicate a missing honeypot
	// feature.
	LogInvalidInvocation(err error)

	// Record when credentials are used by the attacker.
	LogCreds(*logger.Credentials)

	// Get a unique path in the downloads folder that the session can write a
	// file to.
	DownloadPath(source string) string

	// Now is the current honeypot time.
	Now() time.Time
}
// /proc/sys/kernel/{ostype, hostname, osrelease, version, domainname}.
