package vos

import (
	"github.com/spf13/afero"
	"josephlewis.net/osshit/third_party/cowfs"
)

type TenantOS struct {
	sharedOS *SharedOS

	// fs contains a tenant's view of the shared OS.
	fs VFS
}

func NewTenantOS(sharedOS *SharedOS) *TenantOS {
	lfsMemfs := NewLinkingFs(afero.NewMemMapFs())
	ufs := cowfs.NewCopyOnWriteFs(sharedOS.ReadOnlyFs(), lfsMemfs)

	return &TenantOS{
		sharedOS: sharedOS,
		fs:       ufs,
	}
}

// Hostname implements VOS.Hostname.
func (t *TenantOS) Hostname() (string, error) {
	return t.sharedOS.Hostname(), nil
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
