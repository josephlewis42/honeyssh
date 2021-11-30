package vos

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

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
