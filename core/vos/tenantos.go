package vos

import (
	"github.com/spf13/afero"
	"josephlewis.net/osshit/core/logger"
	"josephlewis.net/osshit/third_party/cowfs"
)

type TenantOS struct {
	sharedOS *SharedOS

	// fs contains a tenant's view of the shared OS.
	fs VFS
	// eventRecorder logs events.
	eventRecorder EventRecorder
	// Connected terminal information.
	pty PTY
}

type EventRecorder interface {
	Record(event logger.LogType) error
}

func NewTenantOS(sharedOS *SharedOS, eventRecorder EventRecorder) *TenantOS {
	lfsMemfs := NewLinkingFs(afero.NewMemMapFs())
	ufs := cowfs.NewCopyOnWriteFs(sharedOS.ReadOnlyFs(), lfsMemfs)

	return &TenantOS{
		sharedOS:      sharedOS,
		fs:            ufs,
		eventRecorder: eventRecorder,
	}
}

// Hostname implements VOS.Hostname.
func (t *TenantOS) Hostname() (string, error) {
	return t.sharedOS.Hostname(), nil
}

func (t *TenantOS) SetPTY(pty PTY) {
	t.eventRecorder.Record(&logger.LogEntry_TerminalUpdate{
		TerminalUpdate: &logger.TerminalUpdate{
			Width:  int32(pty.Width),
			Height: int32(pty.Height),
			Term:   pty.Term,
			IsPty:  pty.IsPTY,
		},
	})

	t.pty = pty
}

func (t *TenantOS) GetPTY() PTY {
	return t.pty
}

func (t *TenantOS) InitProc() *TenantProcOS {
	return &TenantProcOS{
		TenantOS:       t,
		VFS:            t.fs,
		VIO:            NewNullIO(),
		VEnv:           NewMapEnv(),
		ExecutablePath: "/sbin/init",
		ProcArgs:       []string{"/sbin/init"},
		PID:            0,
		UID:            0,
		Dir:            "/",
	}
}
