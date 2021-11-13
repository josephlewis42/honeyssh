package vos

import (
	"archive/tar"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/afero"
	"josephlewis.net/osshit/third_party/cowfs"
	"josephlewis.net/osshit/third_party/realpath"
	"josephlewis.net/osshit/third_party/tarfs"
)

// VFS implements a virtual filesystem and is the second layer of the virtual OS.
type VFS = afero.Fs

var nopFs = afero.NewReadOnlyFs(afero.NewMemMapFs())

func NewNopFs() VFS {
	return nopFs
}

func NewCopyOnWriteFs(tarReader *tar.Reader) VFS {
	base := tarfs.New(tarReader)
	roBase := afero.NewReadOnlyFs(base)

	memFs := afero.NewMemMapFs()
	lfsMemfs := NewLinkingFs(memFs)
	ufs := cowfs.NewCopyOnWriteFs(roBase, lfsMemfs)

	return NewLoggingFs(ufs, "union")
}

func NewLoggingFs(base VFS, fsName string) VFS {
	return NewPathMappingFs(base, func(op FsOp, name string) (string, error) {
		fmt.Printf("FS %s %s(%q)\n", fsName, op, name)
		return name, nil
	})
}

func NewSymlinkResolvingRelativeFs(base VFS, Getwd func() (dir string, err error)) VFS {
	rpos := &realpathOs{Getwd, base}
	return NewPathMappingFs(base, func(op FsOp, name string) (string, error) {
		wd, _ := Getwd()
		fmt.Printf("FS operation %s(%q) cwd: %q\n", op, name, wd)
		return realpath.Realpath(rpos, name)
	})
}

type realpathOs struct {
	getwd func() (dir string, err error)
	base  VFS
}

var _ realpath.OS = (*realpathOs)(nil)

func (r *realpathOs) Getwd() (string, error) {
	return r.getwd()
}

func (r *realpathOs) Lstat(name string) (fs.FileInfo, error) {
	if lstater, ok := r.base.(afero.Lstater); ok {
		stat, _, err := lstater.LstatIfPossible(name)
		return stat, err
	}
	return r.base.Stat(name)
}

func (r *realpathOs) Readlink(name string) (string, error) {
	if reader, ok := r.base.(afero.LinkReader); ok {
		return reader.ReadlinkIfPossible(name)
	}
	return "", errors.New("not a link")
}

// LinkingFsWrapper backfills POSIX style symlink functionality onto other file types.
type LinkingFsWrapper struct {
	VFS
}

func NewLinkingFs(base VFS) VFS {
	return &LinkingFsWrapper{base}
}

var _ afero.Symlinker = (*LinkingFsWrapper)(nil)

func (lfs *LinkingFsWrapper) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	fi, err := lfs.VFS.Stat(name)
	return fi, true, err
}

func (lfs *LinkingFsWrapper) ReadlinkIfPossible(name string) (string, error) {
	fi, _, err := lfs.LstatIfPossible(name)
	if err != nil {
		return "", err
	}

	if fi.Mode()&os.ModeSymlink == 0 {
		return "", errors.New("not a link")
	}

	contents, err := afero.ReadFile(lfs.VFS, name)
	return string(contents), err
}

func (lfs *LinkingFsWrapper) SymlinkIfPossible(oldname, newname string) error {
	return afero.WriteFile(lfs.VFS, newname, ([]byte)(oldname), 0666|os.ModeSymlink)
}
