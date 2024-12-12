package vos

import (
	"sync/atomic"
	"time"

	"github.com/josephlewis42/honeyssh/core/config"
	"github.com/spf13/afero"
)

// ProcessFunc is a "process" that can be run.
type ProcessFunc func(VOS) int

// ProcessResolver looks up a fake process by path, it reuturns nil if
// no process was found.
type ProcessResolver func(path string) ProcessFunc

type TimeSource func() time.Time

func NewSharedOS(baseFS VFS, procResolver ProcessResolver, config *config.Configuration, timeSource TimeSource) *SharedOS {
	return &SharedOS{
		mockFS:          baseFS,
		mockPID:         0,
		bootTime:        timeSource(),
		processResolver: procResolver,
		config:          config,
		timeSource:      timeSource,
	}
}

// SharedOS is the shared base OS that each honeypot user gets overlaid on.
//
// All public variables and methods no this type are guaranteed to produce
// immutable objects.
type SharedOS struct {
	// mockFS holds the base filesystem that is shared between ALL programs.
	mockFS VFS
	// mockPID contains the next PID of the system.
	mockPID int32
	// The time the system booted.
	bootTime time.Time
	// The resolver for processes.
	processResolver ProcessResolver
	// The user supplied configuration
	config *config.Configuration
	// Timesource for the OS
	timeSource TimeSource
}

func (s *SharedOS) Hostname() string {
	return s.config.Uname.Nodename
}

func (s *SharedOS) Uname() Utsname {
	return Utsname{
		Sysname:    s.config.Uname.KernelName,
		Nodename:   s.config.Uname.Nodename,
		Release:    s.config.Uname.KernelRelease,
		Version:    s.config.Uname.KernelVersion,
		Machine:    s.config.Uname.HardwarePlatform,
		Domainname: s.config.Uname.Domainname,
	}
}

func (s *SharedOS) BootTime() time.Time {
	return s.bootTime
}

func (s *SharedOS) Now() time.Time {
	return s.timeSource()
}

// ReadOnlyFs returns a read only version of the base filesystem that multiple
// tenants can read from.
func (s *SharedOS) ReadOnlyFs() VFS {
	return afero.NewReadOnlyFs(s.mockFS)
}

// NextPID gets a monotonically increasing PID.
func (s *SharedOS) NextPID() int {
	return int(atomic.AddInt32(&s.mockPID, 1))
}

func (s *SharedOS) SetPID(pid int32) {
	atomic.StoreInt32(&s.mockPID, pid)
}

func (s *SharedOS) GetUser(username string) (usr config.User, ok bool) {
	for _, usr = range s.config.Users {
		if usr.Username == username {
			return usr, true
		}
	}
	return usr, false
}
