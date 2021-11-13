package vos

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

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
		split := strings.SplitN(e, "=", 2)
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

// NewMapEnv creates a new environment backed by a map.
func NewMapEnv() *MapEnv {
	return &MapEnv{}
}

// NewMapEnvFrom creates a new environment with a copy of the environment
// variables in the original environment.
func NewMapEnvFrom(src EnvironFetcher) *MapEnv {
	return NewMapEnvFromEnvList(src.Environ())
}

func NewMapEnvFromEnvList(environ []string) *MapEnv {
	out := &MapEnv{}

	for _, e := range environ {
		split := strings.SplitN(e, "=", 2)
		key, value := split[0], ""
		if len(split) > 1 {
			value = split[1]
		}
		// Ignore error, it will never be set for MapEnv.
		_ = out.Setenv(key, value)
	}

	return out
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
