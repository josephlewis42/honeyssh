package tarfs

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/spf13/afero"
)

type File struct {
	h    tar.Header
	data []byte
	fs   *Fs
}

func (f *File) OpenCursor() *FileCursor {
	return &FileCursor{
		File: f,

		cursor: bytes.NewReader(f.data),
		closed: false,
	}
}

type FileCursor struct {
	*File

	cursor *bytes.Reader
	closed bool
}

func (f *FileCursor) Close() error {
	if f.closed {
		return afero.ErrFileClosed
	}

	f.closed = true
	f.cursor = nil

	return nil
}

func (f *FileCursor) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}

	if f.h.Typeflag == tar.TypeDir {
		return 0, syscall.EISDIR
	}

	return f.cursor.Read(p)
}

func (f *FileCursor) ReadAt(p []byte, off int64) (n int, err error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}

	if f.h.Typeflag == tar.TypeDir {
		return 0, syscall.EISDIR
	}

	return f.cursor.ReadAt(p, off)
}

func (f *FileCursor) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return 0, afero.ErrFileClosed
	}

	if f.h.Typeflag == tar.TypeDir {
		return 0, syscall.EISDIR
	}

	return f.cursor.Seek(offset, whence)
}

func (f *FileCursor) Write(p []byte) (n int, err error) { return 0, syscall.EROFS }

func (f *FileCursor) WriteAt(p []byte, off int64) (n int, err error) { return 0, syscall.EROFS }

func (f *FileCursor) Name() string {
	return filepath.Join(splitpath(f.h.Name))
}

func (f *FileCursor) getDirectoryNames() ([]string, error) {
	var names []string
	for n := range f.fs.files[f.Name()] {
		names = append(names, n)
	}
	sort.Strings(names)

	return names, nil
}

func (f *FileCursor) Readdir(count int) ([]os.FileInfo, error) {
	if f.closed {
		return nil, afero.ErrFileClosed
	}

	if !f.h.FileInfo().IsDir() {
		return nil, syscall.ENOTDIR
	}

	names, err := f.getDirectoryNames()
	if err != nil {
		return nil, err
	}

	d := f.fs.files[f.Name()]
	var fi []os.FileInfo
	for _, n := range names {
		if n == "" {
			continue
		}

		f := d[n]
		fi = append(fi, f.h.FileInfo())
		if count > 0 && len(fi) >= count {
			break
		}
	}

	return fi, nil
}

func (f *FileCursor) Readdirnames(n int) ([]string, error) {
	fi, err := f.Readdir(n)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, f := range fi {
		names = append(names, f.Name())
	}

	return names, nil
}

func (f *FileCursor) Stat() (os.FileInfo, error) { return f.h.FileInfo(), nil }

func (f *FileCursor) Sync() error { return nil }

func (f *FileCursor) Truncate(size int64) error { return syscall.EROFS }

func (f *FileCursor) WriteString(s string) (ret int, err error) { return 0, syscall.EROFS }
