package core

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path"
	"strings"
	"sync"
)

// VOS provides a virtual OS interface.
type VOS interface {
	VIO
	VEnv
	VFS

	// Returns the path to the executable that started the process.
	Executable() (string, error)

	// Getpid returns the process id of the caller.
	Getpid() int

	// Getuid returns the numeric user id of the caller.
	Getuid() int

	// Returns the arguments to the current process.
	Args() []string

	// Getwd returns a rooted path name corresponding to the current directory.
	Getwd() (dir string, err error)

	// Chdir changes the directory.
	Chdir(dir string) error

	// Hostname returns the host name reported by the kernel.
	Hostname() (string, error)
}

// VEnv represents a virtual environment.
type VEnv interface {
	// UserHomeDir returns the current user's home directory.
	UserHomeDir() (string, error)

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

	// ExpandEnv replaces ${var} or $var in the string according to the values of
	// the current environment variables. References to undefined variables are
	// replaced by the empty string.
	ExpandEnv(s string) string

	// Environ returns a copy of strings representing the environment, in the
	// form "key=value".
	Environ() []string

	// Clearenv deletes all environment variables.
	Clearenv()
}

type EnvironFetcher interface {
	// Environ returns a copy of strings representing the environment, in the
	// form "key=value".
	Environ() []string
}

// CopyEnv copies all the environment variables from src to dst.
func CopyEnv(dst VEnv, src EnvironFetcher) error {
	for _, e := range src.Environ() {
		split := strings.SplitN(e, "=", 1)
		key, value := split[0], ""
		if len(split) > 1 {
			value = split[1]
		}
		if err := dst.Setenv(key, value); err != nil {
			return err
		}
	}

	return nil
}

// MapEnv implemnts an in-memory VEnv.
type MapEnv struct {
	rw  sync.RWMutex
	env map[string]string
}

var _ VEnv = (*MapEnv)(nil)

// UserHomeDir implements VEnv.UserHomeDir.
func (m *MapEnv) UserHomeDir() (string, error) {
	return m.Getenv("HOME"), nil
}

// Unsetenv implements VEnv.Unsetenv.
func (m *MapEnv) Unsetenv(key string) error {
	m.rw.Lock()
	defer m.rw.Unlock()
	if m.env != nil {
		delete(m.env, key)
	}
	return nil
}

// Setenv implements VEnv.Setenv.
func (m *MapEnv) Setenv(key, value string) error {
	m.rw.Lock()
	defer m.rw.Unlock()

	if m.env == nil {
		m.env = make(map[string]string)
	}
	m.env[key] = value
	return nil
}

// LookupEnv implements VEnv.LookupEnv.
func (m *MapEnv) LookupEnv(key string) (string, bool) {
	m.rw.RLock()
	defer m.rw.RUnlock()

	val, ok := m.env[key]
	return val, ok
}

// Getenv implements VEnv.Getenv.
func (m *MapEnv) Getenv(key string) string {
	val, _ := m.LookupEnv(key)
	return val
}

// ExpandEnv implements VEnv.ExpandEnv.
func (m *MapEnv) ExpandEnv(s string) string {
	return os.Expand(s, m.Getenv)
}

// Environ implements VEnv.Environ.
func (m *MapEnv) Environ() []string {
	var env []string

	for k, v := range m.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// Clearenv implements VEnv.Clearenv.
func (m *MapEnv) Clearenv() {
	m.rw.Lock()
	defer m.rw.Unlock()
	m.env = make(map[string]string)
}

type VIO interface {
	Stdin() io.ReadCloser
	Stdout() io.WriteCloser
	Stderr() io.WriteCloser
}

func NewVIOAdapter(stdin io.ReadCloser, stdout, stderr io.WriteCloser) *VIOAdapter {
	return &VIOAdapter{
		IStdin:  stdin,
		IStdout: stdout,
		IStderr: stderr,
	}
}

func NewNullIO() VIO {
	return NewVIOAdapter(&ClosedReader{}, &NopWriteCloser{}, &NopWriteCloser{})
}

type VIOAdapter struct {
	IStdin  io.ReadCloser
	IStdout io.WriteCloser
	IStderr io.WriteCloser
}

var _ VIO = (*VIOAdapter)(nil)

func (pr *VIOAdapter) Stdin() io.ReadCloser {
	return pr.IStdin
}

func (pr *VIOAdapter) Stdout() io.WriteCloser {
	return pr.IStdout
}

func (pr *VIOAdapter) Stderr() io.WriteCloser {
	return pr.IStderr
}

// ClosedReader implemnets io.Reader and always throws ErrClosed on Read.
type ClosedReader struct{}

var _ io.ReadCloser = (*ClosedReader)(nil)

func (*ClosedReader) Read([]byte) (int, error) {
	return 0, os.ErrClosed
}

func (*ClosedReader) Close() error {
	return nil
}

type NopWriteCloser struct{}

var _ io.WriteCloser = (*NopWriteCloser)(nil)

func (*NopWriteCloser) Write(b []byte) (int, error) {
	return len(b), nil
}

func (*NopWriteCloser) Close() error {
	return nil
}

type VProc interface {
	// Returns the path to the executable that started the process.
	Executable() (string, error)

	// Getpid returns the process id of the caller.
	Getpid() int

	// Getuid returns the numeric user id of the caller.
	Getuid() int

	// Returns the arguments to the current process.
	Args() []string

	// Getwd returns a rooted path name corresponding to the current directory.
	Getwd() (dir string, err error)

	// Chdir changes the directory.
	Chdir(dir string) error

	// Hostname returns the host name reported by the kernel.
	Hostname() (string, error)
}

type VOSAdapter struct {
	VIO
	VEnv
	VProc
	VFS

	// Args holds command line arguments, including the command as Args[0].
	ProcArgs []string

	// The process ID of the process
	PID int

	// The user ID of the process.
	UID int

	// Dir specifies the working directory of the command.
	Dir string

	// Host specifies the hostname of the system.
	Host string

	// User is (optionally) the cached user.
	User *user.User
}

var _ VOS = (*VOSAdapter)(nil)

// Executable implements VOS.Executable.
func (ea *VOSAdapter) Executable() (string, error) {
	if len(ea.ProcArgs) == 0 {
		return "", os.ErrNotExist
	}

	return ea.ProcArgs[0], nil
}

// Args implements VOS.Args.
func (ea *VOSAdapter) Args() []string {
	return ea.ProcArgs
}

// Getpid implements VOS.Getpid.
func (ea *VOSAdapter) Getpid() int {
	return ea.PID
}

// Getuid implements VOS.Getuid.
func (ea *VOSAdapter) Getuid() int {
	return ea.UID
}

// Getwd implements VOS.Getwd.
func (ea *VOSAdapter) Getwd() (dir string, err error) {
	return ea.Dir, nil
}

// Chdir implements VOS.Chdir.
func (ea *VOSAdapter) Chdir(dir string) (err error) {
	if !path.IsAbs(dir) {
		dir = path.Clean(path.Join(ea.Dir, dir))
	}

	stat, err := ea.VFS.Stat(dir)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %v", dir, err)
	case !stat.IsDir():
		return fmt.Errorf("%s: Not a directory", dir)
	default:
		ea.Dir = dir
		return nil
	}
}

// Hostname implements VOS.Hostname.
func (ea *VOSAdapter) Hostname() (string, error) {
	return ea.Host, nil
}
