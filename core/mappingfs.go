package core

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// FsOp is a textual description of the filesystem operation.
type FsOp = string

const (
	FsOpChtimes  FsOp = "chtimes"
	FsOpSymlink  FsOp = "symlink"
	FsOpChmod    FsOp = "chmod"
	FsOpChown    FsOp = "chown"
	FsOpStat     FsOp = "stat"
	FsOpRename   FsOp = "rename"
	FsOpRemove   FsOp = "remove"
	FsOpOpen     FsOp = "open"
	FsOpMkdir    FsOp = "mkdir"
	FsOpCreate   FsOp = "create"
	FsOpLstat    FsOp = "lstat"
	FsOpReadlink FsOp = "readlink"
)

type FileMapper func(op FsOp, name string) (path string, err error)

// PathMappingFs maps all paths on a filesystem via callback to another path.
type PathMappingFs struct {
	BaseFs afero.Fs
	Mapper FileMapper
}

var _ afero.Lstater = (*PathMappingFs)(nil)

// PathMappingFsFile implements afero.File.
type PathMappingFsFile struct {
	afero.File
	path string
}

// Name returns the name of the file.
func (f *PathMappingFsFile) Name() string {
	sourcename := f.File.Name()
	return strings.TrimPrefix(sourcename, filepath.Clean(f.path))
}

func NewPathMappingFs(base afero.Fs, mapper FileMapper) afero.Fs {
	return &PathMappingFs{BaseFs: base, Mapper: mapper}
}

func (b *PathMappingFs) Chtimes(name string, atime, mtime time.Time) (err error) {
	if name, err = b.Mapper(FsOpChtimes, name); err != nil {
		return &os.PathError{Op: FsOpChtimes, Path: name, Err: err}
	}
	return b.BaseFs.Chtimes(name, atime, mtime)
}

func (b *PathMappingFs) Chmod(name string, mode os.FileMode) (err error) {
	if name, err = b.Mapper(FsOpChmod, name); err != nil {
		return &os.PathError{Op: FsOpChmod, Path: name, Err: err}
	}
	return b.BaseFs.Chmod(name, mode)
}

func (b *PathMappingFs) Chown(name string, uid, gid int) (err error) {
	if name, err = b.Mapper(FsOpChown, name); err != nil {
		return &os.PathError{Op: FsOpChown, Path: name, Err: err}
	}
	return b.BaseFs.Chown(name, uid, gid)
}

func (b *PathMappingFs) Name() string {
	return "PathMappingFs"
}

func (b *PathMappingFs) Stat(name string) (fi os.FileInfo, err error) {
	if name, err = b.Mapper(FsOpStat, name); err != nil {
		return nil, &os.PathError{Op: FsOpStat, Path: name, Err: err}
	}
	return b.BaseFs.Stat(name)
}

func (b *PathMappingFs) Rename(oldname, newname string) (err error) {
	if oldname, err = b.Mapper(FsOpRename, oldname); err != nil {
		return &os.PathError{Op: FsOpRename, Path: oldname, Err: err}
	}
	if newname, err = b.Mapper(FsOpRename, newname); err != nil {
		return &os.PathError{Op: FsOpRename, Path: newname, Err: err}
	}
	return b.BaseFs.Rename(oldname, newname)
}

func (b *PathMappingFs) RemoveAll(name string) (err error) {
	if name, err = b.Mapper(FsOpRemove, name); err != nil {
		return &os.PathError{Op: FsOpRemove, Path: name, Err: err}
	}
	return b.BaseFs.RemoveAll(name)
}

func (b *PathMappingFs) Remove(name string) (err error) {
	if name, err = b.Mapper(FsOpRemove, name); err != nil {
		return &os.PathError{Op: FsOpRemove, Path: name, Err: err}
	}
	return b.BaseFs.Remove(name)
}

func (b *PathMappingFs) OpenFile(name string, flag int, mode os.FileMode) (f afero.File, err error) {
	if name, err = b.Mapper(FsOpOpen, name); err != nil {
		return nil, &os.PathError{Op: FsOpOpen, Path: name, Err: err}
	}
	sourcef, err := b.BaseFs.OpenFile(name, flag, mode)
	if err != nil {
		return nil, err
	}
	return &PathMappingFsFile{sourcef, name}, nil
}

func (b *PathMappingFs) Open(name string) (f afero.File, err error) {
	if name, err = b.Mapper(FsOpOpen, name); err != nil {
		return nil, &os.PathError{Op: FsOpOpen, Path: name, Err: err}
	}
	sourcef, err := b.BaseFs.Open(name)
	if err != nil {
		return nil, err
	}
	return &PathMappingFsFile{File: sourcef, path: name}, nil
}

func (b *PathMappingFs) Mkdir(name string, mode os.FileMode) (err error) {
	if name, err = b.Mapper(FsOpMkdir, name); err != nil {
		return &os.PathError{Op: FsOpMkdir, Path: name, Err: err}
	}
	return b.BaseFs.Mkdir(name, mode)
}

func (b *PathMappingFs) MkdirAll(name string, mode os.FileMode) (err error) {
	if name, err = b.Mapper(FsOpMkdir, name); err != nil {
		return &os.PathError{Op: FsOpMkdir, Path: name, Err: err}
	}
	return b.BaseFs.MkdirAll(name, mode)
}

func (b *PathMappingFs) Create(name string) (f afero.File, err error) {
	if name, err = b.Mapper(FsOpCreate, name); err != nil {
		return nil, &os.PathError{Op: FsOpCreate, Path: name, Err: err}
	}
	sourcef, err := b.BaseFs.Create(name)
	if err != nil {
		return nil, err
	}
	return &PathMappingFsFile{File: sourcef, path: name}, nil
}

func (b *PathMappingFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	name, err := b.Mapper(FsOpLstat, name)
	if err != nil {
		return nil, false, &os.PathError{Op: FsOpLstat, Path: name, Err: err}
	}
	if lstater, ok := b.BaseFs.(afero.Lstater); ok {
		return lstater.LstatIfPossible(name)
	}
	fi, err := b.BaseFs.Stat(name)
	return fi, false, err
}

func (b *PathMappingFs) SymlinkIfPossible(oldname, newname string) error {
	oldname, err := b.Mapper(FsOpSymlink, oldname)
	if err != nil {
		return &os.LinkError{Op: FsOpSymlink, Old: oldname, New: newname, Err: err}
	}
	newname, err = b.Mapper(FsOpSymlink, newname)
	if err != nil {
		return &os.LinkError{Op: FsOpSymlink, Old: oldname, New: newname, Err: err}
	}
	if linker, ok := b.BaseFs.(afero.Linker); ok {
		return linker.SymlinkIfPossible(oldname, newname)
	}
	return &os.LinkError{Op: FsOpSymlink, Old: oldname, New: newname, Err: afero.ErrNoSymlink}
}

func (b *PathMappingFs) ReadlinkIfPossible(name string) (string, error) {
	name, err := b.Mapper(FsOpReadlink, name)
	if err != nil {
		return "", &os.PathError{Op: FsOpReadlink, Path: name, Err: err}
	}
	if reader, ok := b.BaseFs.(afero.LinkReader); ok {
		return reader.ReadlinkIfPossible(name)
	}
	return "", &os.PathError{Op: FsOpReadlink, Path: name, Err: afero.ErrNoReadlink}
}
