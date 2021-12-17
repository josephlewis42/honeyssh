package vos

import (
	"fmt"
	"strings"
	"sync"
)

// CopyEnv copies all the environment variables from src to dst.
func CopyEnv(dst VEnv, src []string) error {
	for _, e := range src {
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

type EnvironFetcher interface {
	// Environ returns a copy of strings representing the environment, in the
	// form "key=value".
	Environ() []string
}

// NewMapEnvFrom creates a new environment with a copy of the environment
// variables in the original environment.
func NewMapEnvFrom(src EnvironFetcher) *MapEnv {
	return NewMapEnvFromEnvList(src.Environ())
}

func NewMapEnvFromEnvList(environ []string) *MapEnv {
	out := &MapEnv{}

	// Ignore error, it will never be set for MapEnv.
	_ = CopyEnv(out, environ)

	return out
}

// MapEnv implemnts an in-memory VEnv.
type MapEnv struct {
	rw  sync.RWMutex
	env map[string]string
}

var _ VEnv = (*MapEnv)(nil)

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

// Environ implements VEnv.Environ.
func (m *MapEnv) Environ() []string {
	var env []string

	for k, v := range m.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}
