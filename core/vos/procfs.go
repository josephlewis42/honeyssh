package vos

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/spf13/afero"
	"josephlewis.net/osshit/third_party/memmapfs/mem"
)

type procFile struct {
	Name      string
	Generator func(vos *SharedOS) string
}

var procFiles = []procFile{
	{Name: "/cpuinfo", Generator: func(vos *SharedOS) string {
		// Copied from gVisor:
		// https://github.com/google/gvisor/blob/master/pkg/sentry/fs/proc/README.md
		return `processor   : 0
vendor_id   : GenuineIntel
cpu family  : 6
model       : 45
model name  : unknown
stepping    : unknown
cpu MHz     : 1234.588
fpu     : yes
fpu_exception   : yes
cpuid level : 13
wp      : yes
flags       : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush dts acpi mmx fxsr sse sse2 ss ht tm pbe syscall nx pdpe1gb rdtscp lm pni pclmulqdq dtes64 monitor ds_cpl vmx smx est tm2 ssse3 cx16 xtpr pdcm pcid dca sse4_1 sse4_2 x2apic popcnt tsc_deadline_timer aes xsave avx xsaveopt
bogomips    : 1234.59
clflush size    : 64
cache_alignment : 64
address sizes   : 46 bits physical, 48 bits virtual
`
	}},
	{Name: "/uptime", Generator: func(vos *SharedOS) string {
		uptime := vos.Now().Sub(vos.BootTime()).Seconds()
		// [seconds running] [seconds idle]
		return fmt.Sprintf("%0.2f 0.00\n", uptime)
	}},
	{Name: "/version", Generator: func(vos *SharedOS) string {
		uname := vos.Uname()
		return fmt.Sprintf("%s %s %s\n", uname.Sysname, uname.Release, uname.Version)
	}},
}

func resolveProcFile(name string, vos *SharedOS) (afero.File, error) {
	for _, procFile := range procFiles {
		if procFile.Name == name {
			file := mem.CreateFile(name, vos.Now)
			mem.NewFileHandle(file).WriteString(procFile.Generator(vos))
			return mem.NewReadOnlyFileHandle(file), nil
		}
	}

	if name == "/" {
		dir := mem.CreateDir(name, vos.Now)
		for _, procFile := range procFiles {
			mem.AddToMemDir(dir, mem.CreateFile(procFile.Name, vos.Now))
		}
		return mem.NewReadOnlyFileHandle(dir), nil

	}

	return nil, fs.ErrNotExist
}

func NewProcFS(sharedOS *SharedOS) *ProcFS {
	return &ProcFS{sharedOS: sharedOS}
}

type ProcFS struct {
	sharedOS *SharedOS
	VirtualFS
}

var _ VFS = (*ProcFS)(nil)

func (pfs *ProcFS) OpenFile(name string, flag int, perm fs.FileMode) (afero.File, error) {
	return resolveProcFile(name, pfs.sharedOS)
}

func (pfs *ProcFS) Open(name string) (afero.File, error) {
	return resolveProcFile(name, pfs.sharedOS)
}

func (*ProcFS) Name() string {
	return "/proc"
}

func (pfs *ProcFS) Stat(name string) (fs.FileInfo, error) {
	fd, err := resolveProcFile(name, pfs.sharedOS)
	if err != nil {
		return nil, err
	}
	return fd.Stat()
}

// VirtualFS returns ErrNotExist for any write or modify operations.
type VirtualFS struct{}

func (*VirtualFS) Rename(oldname, newname string) error {
	return fs.ErrNotExist
}

func (*VirtualFS) RemoveAll(name string) error {
	return fs.ErrNotExist
}

func (*VirtualFS) Remove(name string) error {
	return fs.ErrNotExist
}

func (*VirtualFS) MkdirAll(_ string, _ fs.FileMode) error {
	return fs.ErrNotExist
}

func (*VirtualFS) Mkdir(_ string, _ fs.FileMode) error {
	return fs.ErrNotExist
}

func (*VirtualFS) Create(_ string) (afero.File, error) {
	return nil, fs.ErrNotExist
}

func (*VirtualFS) Chtimes(_ string, _, _ time.Time) error {
	return fs.ErrNotExist
}

func (*VirtualFS) Chown(_ string, _ int, _ int) error {
	return fs.ErrNotExist
}

func (*VirtualFS) Chmod(_ string, _ fs.FileMode) error {
	return fs.ErrNotExist
}
