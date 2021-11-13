package vos

import (
	"sync/atomic"
	"time"

	"github.com/spf13/afero"
)

func NewSharedOS(baseFS VFS, hostname string) *SharedOS {
	return &SharedOS{
		mockFS:       baseFS,
		mockHostname: hostname,
		mockPID:      0,
		bootTime:     time.Now(),
	}
}

// SharedOS is the shared base OS that each honeypot user gets overlaid on.
//
// All public variables and methods no this type are guaranteed to produce
// immutable objects.
type SharedOS struct {
	// mockFS holds the base filesystem that is shared between ALL programs.
	mockFS VFS

	// mockHostname holds the displayed hostname of the OS.
	mockHostname string

	// mockPID contains the next PID of the system.
	mockPID int32

	// The time the system booted.
	bootTime time.Time
}

func (s *SharedOS) Hostname() string {
	return s.mockHostname
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
