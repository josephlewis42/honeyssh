package vos

import (
	"archive/tar"
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"

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

	return ufs
}

func NewSymlinkResolvingRelativeFs(base VFS, Getwd func() (dir string, err error)) VFS {
	rpos := &realpathOs{Getwd, base}
	return NewPathMappingFs(base, func(op FsOp, name string) (string, error) {
		switch op {
		case FsOpMkdir:
			dir, err := realpath.Realpath(rpos, path.Dir(name))
			// Expect at least one not exist, but we'll go as far as possible.
			if err != nil && strings.Contains(err.Error(), "no such file or directory") {
				return dir, err
			}
			return path.Join(dir, path.Base(name)), nil
		default:
			return realpath.Realpath(rpos, name)
		}
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
