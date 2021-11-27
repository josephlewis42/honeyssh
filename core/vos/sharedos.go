package vos

import (
	"sync/atomic"
	"time"

	"github.com/spf13/afero"
	"josephlewis.net/osshit/core/config"
)

// ProcessFunc is a "process" that can be run.
type ProcessFunc func(VOS) int

// ProcessResolver looks up a fake process by path, it reuturns nil if
// no process was found.
type ProcessResolver func(path string) ProcessFunc

func NewSharedOS(baseFS VFS, utsname Utsname, procResolver ProcessResolver, config *config.Configuration) *SharedOS {
	return &SharedOS{
		mockFS:          baseFS,
		mockUtsname:     utsname,
		mockPID:         0,
		bootTime:        time.Now(),
		processResolver: procResolver,
		config:          config,
	}
}

// SharedOS is the shared base OS that each honeypot user gets overlaid on.
//
// All public variables and methods no this type are guaranteed to produce
// immutable objects.
type SharedOS struct {
	// mockFS holds the base filesystem that is shared between ALL programs.
	mockFS VFS
	// mockUtsname holds the displayed OS info including hostname.
	mockUtsname Utsname
	// mockPID contains the next PID of the system.
	mockPID int32
	// The time the system booted.
	bootTime time.Time
	// The resolver for processes.
	processResolver ProcessResolver
	// The user supplied configuration
	config *config.Configuration
}

func (s *SharedOS) Hostname() string {
	return s.mockUtsname.Nodename
}

func (s *SharedOS) Uname() Utsname {
	return s.mockUtsname
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

func (s *SharedOS) SetBootTime(bootTime time.Time) {
	s.bootTime = bootTime
}
