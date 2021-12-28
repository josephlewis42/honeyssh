package vos

import (
	"io"
	"net"
	"os"
	"time"

	"github.com/spf13/afero"
	"github.com/josephlewis42/honeyssh/core/logger"
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
	Hostname() string
	// Uname mimics the uname syscall.
	Uname() Utsname
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

// VEnv represents a virtual environment.
type VEnv interface {
	// Unsetenv unsets a single environment variable.
	Unsetenv(key string) error

	// Setenv sets the value of the environment variable named by the key.
	// It returns an error, if any.
	Setenv(key, value string) error

	// LookupEnv retrieves the value of the environment variable named by the key.
	// If the variable is present in the environment the value (which may be
	// empty) is returned and the boolean is true. Otherwise the returned value
	// will be empty and the boolean will be false.
	LookupEnv(key string) (string, bool)

	// Getenv retrieves the value of the environment variable named by the key.
	// It returns the value, which will be empty if the variable is not present.
	// To distinguish between an empty value and an unset value, use LookupEnv.
	Getenv(key string) string

	// Environ returns a copy of strings representing the environment, in the
	// form "key=value".
	Environ() []string
}

type VIO interface {
	Stdin() io.ReadCloser
	Stdout() io.WriteCloser
	Stderr() io.WriteCloser
}

type VProc interface {
	// Getpid returns the process id of the caller.
	Getpid() int

	// Getuid returns the numeric user id of the caller.
	Getuid() int

	// Returns the arguments to the current process.
	Args() []string

	// Getwd returns a rooted path name corresponding to the current directory.
	Getwd() (dir string)

	// Chdir changes the directory.
	Chdir(dir string) error

	// Run executes the command, waits for it to finish and returns the status
	// code.
	Run() int
}

// VFS implements a virtual filesystem and is the second layer of the virtual OS.
type VFS interface {
	// Create creates a file in the filesystem, returning the file and an
	// error, if any happens.
	Create(name string) (afero.File, error)

	// Mkdir creates a directory in the filesystem, return an error if any
	// happens.
	Mkdir(name string, perm os.FileMode) error

	// MkdirAll creates a directory path and all parents that does not exist
	// yet.
	MkdirAll(path string, perm os.FileMode) error

	// Open opens a file, returning it or an error, if any happens.
	Open(name string) (afero.File, error)

	// OpenFile opens a file using the given flags and the given mode.
	OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)

	// Remove removes a file identified by name, returning an error, if any
	// happens.
	Remove(name string) error

	// RemoveAll removes a directory path and any children it contains. It
	// does not fail if the path does not exist (return nil).
	RemoveAll(path string) error

	// Rename renames a file.
	Rename(oldname, newname string) error

	// Stat returns a FileInfo describing the named file, or an error, if any
	// happens.
	Stat(name string) (os.FileInfo, error)

	Name() string

	// Chmod changes the mode of the named file to mode.
	Chmod(name string, mode os.FileMode) error

	// Chown changes the uid and gid of the named file.
	Chown(name string, uid, gid int) error

	// Chtimes changes the access and modification times of the named file
	Chtimes(name string, atime time.Time, mtime time.Time) error
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
	DownloadPath(source string) (afero.File, error)

	// Now is the current honeypot time.
	Now() time.Time
}

// /proc/sys/kernel/{ostype, hostname, osrelease, version, domainname}.
