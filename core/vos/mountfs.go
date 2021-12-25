package vos

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/spf13/afero"
)

type Mount struct {
	// Path is the directory the volume is mounted at.
	Path string
	FS   VFS
}

func NewMountFS(root VFS) *MountFS {
	return &MountFS{
		Root: root,
	}
}

type MountFS struct {
	// Root is the root filesystem.
	Root VFS
	// List of mounted volumes, sorted deepest first.
	Mounts []Mount
}

func (mfs *MountFS) Mount(path string, mountFS VFS) error {
	path = strings.TrimSuffix(path, "/")

	// TODO: enable this once the root fs works correctly and supports empty dirs.
	// Check that a directory exists for this FS first sore it's accessible.
	// s, err := mfs.Stat(path)
	// switch {
	// case err != nil:
	// 	return fmt.Errorf("invalid mount path %q: %v", path, err)
	// case !s.IsDir():
	// 	return fmt.Errorf("invalid mount path %q: not a directory", path)
	// }

	mfs.Mounts = append(mfs.Mounts, Mount{Path: path, FS: mountFS})
	sort.Slice(mfs.Mounts, func(i, j int) bool {
		return len(mfs.Mounts[i].Path) > len(mfs.Mounts[j].Path)
	})

	return nil
}

func (mfs *MountFS) Resolve(path string) (VFS, string) {
	path = strings.TrimSuffix(path, "/")

	for _, mount := range mfs.Mounts {
		// The mount matches if the path is the same, or if path falls under the
		// mount point.
		if path == mount.Path || strings.HasPrefix(path, mount.Path+"/") {
			newPrefix := strings.TrimPrefix(path, mount.Path)
			if newPrefix == "" {
				newPrefix = "/"
			}
			return mount.FS, newPrefix
		}
	}

	return mfs.Root, path
}

var _ VFS = (*MountFS)(nil)

func (mfs *MountFS) OpenFile(name string, flag int, perm fs.FileMode) (afero.File, error) {
	vfs, newname := mfs.Resolve(name)
	return vfs.OpenFile(newname, flag, perm)
}

// Open opens a file, returning it or an error, if any happens.
func (mfs *MountFS) Open(name string) (afero.File, error) {
	vfs, newname := mfs.Resolve(name)
	return vfs.Open(newname)
}

func (mfs *MountFS) Name() string {
	return "mount"
}

// Stat returns a FileInfo describing the named file, or an error, if any
// happens.
func (mfs *MountFS) Stat(name string) (fs.FileInfo, error) {
	vfs, newname := mfs.Resolve(name)
	return vfs.Stat(newname)
}

// Rename renames (moves) oldpath to newpath. If newpath already exists and is
// not a directory, Rename replaces it. Files may not be moved across FS
// boundaries.
func (mfs *MountFS) Rename(oldname, newname string) error {
	ovfs, newoldname := mfs.Resolve(oldname)
	nvfs, newnewname := mfs.Resolve(newname)

	if ovfs != nvfs {
		return fmt.Errorf("stopping at filesystem boundary")
	}

	return ovfs.Rename(newoldname, newnewname)
}

// RemoveAll removes a directory path and any children it contains. It
// does not fail if the path does not exist (return nil).
func (mfs *MountFS) RemoveAll(name string) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.RemoveAll(newname)
}

// Remove removes a file identified by name, returning an error, if any happens.
func (mfs *MountFS) Remove(name string) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.Remove(newname)
}

// MkdirAll creates a directory path and all parents that does not exist yet.
func (mfs *MountFS) MkdirAll(name string, mode fs.FileMode) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.MkdirAll(newname, mode)
}

// Mkdir creates a directory in the filesystem, return an error if any happens.
func (mfs *MountFS) Mkdir(name string, mode fs.FileMode) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.Mkdir(newname, mode)
}

// Create creates a file in the filesystem, returning the file and an
// error, if any happens.
func (mfs *MountFS) Create(name string) (afero.File, error) {
	vfs, newname := mfs.Resolve(name)
	return vfs.Create(newname)
}

// Chtimes changes the access and modification times of the named file
func (mfs *MountFS) Chtimes(name string, atime, mtime time.Time) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.Chtimes(newname, atime, mtime)
}

// Chown changes the uid and gid of the named file.
func (mfs *MountFS) Chown(name string, uid, gid int) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.Chown(newname, uid, gid)
}

// Chmod changes the mode of the named file to mode.
func (mfs *MountFS) Chmod(name string, mode fs.FileMode) error {
	vfs, newname := mfs.Resolve(name)
	return vfs.Chmod(newname, mode)
}
