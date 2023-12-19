package vfs

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/spf13/afero"
	"github.com/tetratelabs/wazero/experimental/sys"
)

type AferoAdapter struct {
	fs      *Memory
	context VFSContext
}

var _ afero.Fs = (*AferoAdapter)(nil)

func (adapter *AferoAdapter) adaptFile(handle *inodeHandle, err sys.Errno) (afero.File, error) {
	if err != Success {
		return nil, err
	}

	return &AferoFileAdapter{
		fs:      adapter.fs,
		context: adapter.context,
		handle:  handle,
	}, nil
}

// Create creates a file in the filesystem, returning the file and an
// error, if any happens.
func (adapter *AferoAdapter) Create(name string) (afero.File, error) {
	return adapter.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
}

// Open opens a file, returning it or an error, if any happens.
func (adapter *AferoAdapter) Open(name string) (afero.File, error) {
	return adapter.OpenFile(name, os.O_RDONLY, 0)
}

// MkdirAll creates a directory path and all parents that does not exist
// yet.
func (adapter *AferoAdapter) MkdirAll(name string, perm os.FileMode) error {
	var soFar []string
	for _, part := range strings.Split(name, "/") {
		soFar = append(soFar, part)

		err := adapter.Mkdir(path.Join(soFar...), perm)
		if err == nil || err == fs.ErrExist {
			continue
		} else {
			return err
		}
	}
	return nil
}

// Mkdir creates a directory in the filesystem, return an error if any
// happens.
func (adapter *AferoAdapter) Mkdir(name string, perm os.FileMode) error {
	return adapter.fs.Mkdir(adapter.context, name, perm)
}

// OpenFile opens a file using the given flags and the given mode.
func (adapter *AferoAdapter) OpenFile(name string, flags int, perm os.FileMode) (afero.File, error) {
	return adapter.adaptFile(adapter.fs.Open(adapter.context, name, flags, perm))
}

// Remove removes a file identified by name, returning an error, if any
// happens.
func (adapter *AferoAdapter) Remove(name string) error {
	return errors.New("not supported")
}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (adapter *AferoAdapter) RemoveAll(path string) error {
	return errors.New("not supported")
}

// Rename renames a file.
func (adapter *AferoAdapter) Rename(oldname, newname string) error {
	return errors.New("not supported")
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (adapter *AferoAdapter) Stat(name string) (os.FileInfo, error) {
	return nil, errors.New("not supported")
}

// The name of this FileSystem
func (adapter *AferoAdapter) Name() string {
	return "AferoAdapter"
}

// Chmod changes the mode of the named file to mode.
func (adapter *AferoAdapter) Chmod(name string, mode os.FileMode) error {
	return nil

}

// Chown changes the uid and gid of the named file.
func (adapter *AferoAdapter) Chown(name string, uid, gid int) error {
	return nil
}

//Chtimes changes the access and modification times of the named file
func (adapter *AferoAdapter) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return nil
}

type AferoFileAdapter struct {
	fs      *Memory
	context VFSContext
	handle  *inodeHandle
}

var _ afero.File = (*AferoFileAdapter)(nil)

// Close implements afero.File
func (adapter *AferoFileAdapter) Close() error {
	panic("unimplemented")
}

// Read implements afero.File
func (adapter *AferoFileAdapter) Read(p []byte) (n int, err error) {
	panic("unimplemented")
}

// ReadAt implements afero.File
func (adapter *AferoFileAdapter) ReadAt(p []byte, off int64) (n int, err error) {
	panic("unimplemented")
}

// Seek implements afero.File
func (adapter *AferoFileAdapter) Seek(offset int64, whence int) (int64, error) {
	panic("unimplemented")
}

// Write implements afero.File
func (adapter *AferoFileAdapter) Write(p []byte) (n int, err error) {
	panic("unimplemented")
}

// WriteAt implements afero.File
func (adapter *AferoFileAdapter) WriteAt(p []byte, off int64) (n int, err error) {
	panic("unimplemented")
}

// Name implements afero.File
func (adapter *AferoFileAdapter) Name() string {
	panic("unimplemented")
}

// Readdir implements afero.File
func (adapter *AferoFileAdapter) Readdir(count int) ([]fs.FileInfo, error) {
	panic("unimplemented")
}

// Readdirnames implements afero.File
func (adapter *AferoFileAdapter) Readdirnames(n int) ([]string, error) {
	panic("unimplemented")
}

// Stat implements afero.File
func (adapter *AferoFileAdapter) Stat() (fs.FileInfo, error) {
	panic("unimplemented")
}

// Sync implements afero.File
func (adapter *AferoFileAdapter) Sync() error {
	panic("unimplemented")
}

// Truncate implements afero.File
func (adapter *AferoFileAdapter) Truncate(size int64) error {
	return adapter.handle.Truncate(int(size))
}

// WriteString implements afero.File
func (adapter *AferoFileAdapter) WriteString(s string) (ret int, err error) {
	panic("unimplemented")
}
