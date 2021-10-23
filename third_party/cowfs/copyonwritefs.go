package cowfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/afero"
)

var _ afero.Lstater = (*CopyOnWriteFs)(nil)

// The CopyOnWriteFs is a union filesystem: a read only base file system with
// a possibly writeable layer on top. Changes to the file system will only
// be made in the overlay: Changing an existing file in the base layer which
// is not present in the overlay will copy the file to the overlay ("changing"
// includes also calls to e.g. Chtimes(), Chmod() and Chown()).
//
// Reading directories is currently only supported via Open(), not OpenFile().
type CopyOnWriteFs struct {
	base  afero.Fs
	layer afero.Fs
}

func NewCopyOnWriteFs(base afero.Fs, layer afero.Fs) afero.Fs {
	return &CopyOnWriteFs{base: base, layer: layer}
}

// Returns true if the file is not in the overlay
func (u *CopyOnWriteFs) isBaseFile(name string) (bool, error) {
	if _, err := u.layer.Stat(name); err == nil {
		return false, nil
	}
	_, err := u.base.Stat(name)
	if err != nil {
		if oerr, ok := err.(*os.PathError); ok {
			if oerr.Err == os.ErrNotExist || oerr.Err == syscall.ENOENT || oerr.Err == syscall.ENOTDIR {
				return false, nil
			}
		}
		if err == syscall.ENOENT {
			return false, nil
		}
	}
	return true, err
}

func (u *CopyOnWriteFs) copyToLayer(name string) error {
	return copyToLayer(u.base, u.layer, name)
}

func copyToLayer(base afero.Fs, layer afero.Fs, name string) error {
	bfh, err := base.Open(name)
	if err != nil {
		return err
	}
	defer bfh.Close()

	// First make sure the directory exists
	exists, err := afero.Exists(layer, filepath.Dir(name))
	if err != nil {
		return err
	}
	if !exists {
		err = layer.MkdirAll(filepath.Dir(name), 0777) // FIXME?
		if err != nil {
			return err
		}
	}

	// Create the file on the overlay
	lfh, err := layer.Create(name)
	if err != nil {
		return err
	}
	n, err := io.Copy(lfh, bfh)
	if err != nil {
		// If anything fails, clean up the file
		layer.Remove(name)
		lfh.Close()
		return err
	}

	bfi, err := bfh.Stat()
	if err != nil || bfi.Size() != n {
		layer.Remove(name)
		lfh.Close()
		return syscall.EIO
	}

	err = lfh.Close()
	if err != nil {
		layer.Remove(name)
		lfh.Close()
		return err
	}
	return layer.Chtimes(name, bfi.ModTime(), bfi.ModTime())
}

func (u *CopyOnWriteFs) Chtimes(name string, atime, mtime time.Time) error {
	b, err := u.isBaseFile(name)
	if err != nil {
		return err
	}
	if b {
		if err := u.copyToLayer(name); err != nil {
			return err
		}
	}
	return u.layer.Chtimes(name, atime, mtime)
}

func (u *CopyOnWriteFs) Chmod(name string, mode os.FileMode) error {
	b, err := u.isBaseFile(name)
	if err != nil {
		return err
	}
	if b {
		if err := u.copyToLayer(name); err != nil {
			return err
		}
	}
	return u.layer.Chmod(name, mode)
}

func (u *CopyOnWriteFs) Chown(name string, uid, gid int) error {
	b, err := u.isBaseFile(name)
	if err != nil {
		return err
	}
	if b {
		if err := u.copyToLayer(name); err != nil {
			return err
		}
	}
	return u.layer.Chown(name, uid, gid)
}

func (u *CopyOnWriteFs) Stat(name string) (os.FileInfo, error) {
	fi, err := u.layer.Stat(name)
	if err != nil {
		isNotExist := u.isNotExist(err)
		if isNotExist {
			return u.base.Stat(name)
		}
		return nil, err
	}
	return fi, nil
}

func (u *CopyOnWriteFs) LstatIfPossible(name string) (os.FileInfo, bool, error) {
	llayer, ok1 := u.layer.(afero.Lstater)
	lbase, ok2 := u.base.(afero.Lstater)

	if ok1 {
		fi, b, err := llayer.LstatIfPossible(name)
		if err == nil {
			return fi, b, nil
		}

		if !u.isNotExist(err) {
			return nil, b, err
		}
	}

	if ok2 {
		fi, b, err := lbase.LstatIfPossible(name)
		if err == nil {
			return fi, b, nil
		}
		if !u.isNotExist(err) {
			return nil, b, err
		}
	}

	fi, err := u.Stat(name)

	return fi, false, err
}

func (u *CopyOnWriteFs) SymlinkIfPossible(oldname, newname string) error {
	if slayer, ok := u.layer.(afero.Linker); ok {
		return slayer.SymlinkIfPossible(oldname, newname)
	}

	return &os.LinkError{Op: "symlink", Old: oldname, New: newname, Err: afero.ErrNoSymlink}
}

func (u *CopyOnWriteFs) ReadlinkIfPossible(name string) (string, error) {
	rlayer, ok1 := u.layer.(afero.LinkReader)
	rbase, ok2 := u.base.(afero.LinkReader)

	if ok1 {
		fi, err := rlayer.ReadlinkIfPossible(name)
		if err == nil {
			return fi, nil
		}

		if !u.isNotExist(err) {
			return "", err
		}
	}

	if ok2 {
		fi, err := rbase.ReadlinkIfPossible(name)
		if err == nil {
			return fi, nil
		}
		if !u.isNotExist(err) {
			return "", err
		}
	}

	return "", &os.PathError{Op: "readlink", Path: name, Err: afero.ErrNoReadlink}
}

func (u *CopyOnWriteFs) isNotExist(err error) bool {
	if e, ok := err.(*os.PathError); ok {
		err = e.Err
	}
	if err == os.ErrNotExist || err == syscall.ENOENT || err == syscall.ENOTDIR {
		return true
	}
	return false
}

// Renaming files present only in the base layer is not permitted
func (u *CopyOnWriteFs) Rename(oldname, newname string) error {
	b, err := u.isBaseFile(oldname)
	if err != nil {
		return err
	}
	if b {
		return syscall.EPERM
	}
	return u.layer.Rename(oldname, newname)
}

// Removing files present only in the base layer is not permitted. If
// a file is present in the base layer and the overlay, only the overlay
// will be removed.
func (u *CopyOnWriteFs) Remove(name string) error {
	err := u.layer.Remove(name)
	switch err {
	case syscall.ENOENT:
		_, err = u.base.Stat(name)
		if err == nil {
			return syscall.EPERM
		}
		return syscall.ENOENT
	default:
		return err
	}
}

func (u *CopyOnWriteFs) RemoveAll(name string) error {
	err := u.layer.RemoveAll(name)
	switch err {
	case syscall.ENOENT:
		_, err = u.base.Stat(name)
		if err == nil {
			return syscall.EPERM
		}
		return syscall.ENOENT
	default:
		return err
	}
}

func (u *CopyOnWriteFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	b, err := u.isBaseFile(name)
	if err != nil {
		return nil, err
	}

	if flag&(os.O_WRONLY|os.O_RDWR|os.O_APPEND|os.O_CREATE|os.O_TRUNC) != 0 {
		if b {
			if err = u.copyToLayer(name); err != nil {
				return nil, err
			}
			return u.layer.OpenFile(name, flag, perm)
		}

		dir := filepath.Dir(name)
		isaDir, err := afero.IsDir(u.base, dir)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		if isaDir {
			if err = u.layer.MkdirAll(dir, 0777); err != nil {
				return nil, err
			}
			return u.layer.OpenFile(name, flag, perm)
		}

		isaDir, err = afero.IsDir(u.layer, dir)
		if err != nil {
			return nil, err
		}
		if isaDir {
			return u.layer.OpenFile(name, flag, perm)
		}

		return nil, &os.PathError{Op: "open", Path: name, Err: syscall.ENOTDIR} // ...or os.ErrNotExist?
	}
	if b {
		return u.base.OpenFile(name, flag, perm)
	}
	return u.layer.OpenFile(name, flag, perm)
}

// This function handles the 9 different possibilities caused
// by the union which are the intersection of the following...
//  layer: doesn't exist, exists as a file, and exists as a directory
//  base:  doesn't exist, exists as a file, and exists as a directory
func (u *CopyOnWriteFs) Open(name string) (afero.File, error) {
	// Since the overlay overrides the base we check that first
	b, err := u.isBaseFile(name)
	if err != nil {
		return nil, err
	}

	// If overlay doesn't exist, return the base (base state irrelevant)
	if b {
		return u.base.Open(name)
	}

	// If overlay is a file, return it (base state irrelevant)
	dir, err := afero.IsDir(u.layer, name)
	if err != nil {
		return nil, err
	}
	if !dir {
		return u.layer.Open(name)
	}

	// Overlay is a directory, base state now matters.
	// Base state has 3 states to check but 2 outcomes:
	// A. It's a file or non-readable in the base (return just the overlay)
	// B. It's an accessible directory in the base (return a UnionFile)

	// If base is file or nonreadable, return overlay
	dir, err = afero.IsDir(u.base, name)
	if !dir || err != nil {
		return u.layer.Open(name)
	}

	// Both base & layer are directories
	// Return union file (if opens are without error)
	bfile, bErr := u.base.Open(name)
	lfile, lErr := u.layer.Open(name)

	// If either have errors at this point something is very wrong. Return nil and the errors
	if bErr != nil || lErr != nil {
		return nil, fmt.Errorf("BaseErr: %v\nOverlayErr: %v", bErr, lErr)
	}

	return &afero.UnionFile{Base: bfile, Layer: lfile}, nil
}

func (u *CopyOnWriteFs) Mkdir(name string, perm os.FileMode) error {
	dir, err := afero.IsDir(u.base, name)
	if err != nil {
		return u.layer.MkdirAll(name, perm)
	}
	if dir {
		return afero.ErrFileExists
	}
	return u.layer.MkdirAll(name, perm)
}

func (u *CopyOnWriteFs) Name() string {
	return "CopyOnWriteFs"
}

func (u *CopyOnWriteFs) MkdirAll(name string, perm os.FileMode) error {
	dir, err := afero.IsDir(u.base, name)
	if err != nil {
		return u.layer.MkdirAll(name, perm)
	}
	if dir {
		// This is in line with how os.MkdirAll behaves.
		return nil
	}
	return u.layer.MkdirAll(name, perm)
}

func (u *CopyOnWriteFs) Create(name string) (afero.File, error) {
	return u.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
}
