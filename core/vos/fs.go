package vos

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"

	"github.com/josephlewis42/honeyssh/core/config"
	"github.com/josephlewis42/honeyssh/third_party/cowfs"
	"github.com/josephlewis42/honeyssh/third_party/memmapfs"
	"github.com/josephlewis42/honeyssh/third_party/realpath"
	"github.com/spf13/afero"
)

func NewVFSFromConfig(configuration *config.Configuration) (VFS, error) {
	fd, err := configuration.OpenFilesystemTarGz()
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	gr, err := gzip.NewReader(fd)
	if err != nil {
		return nil, err
	}

	lfsMemfs := NewLinkingFs(memmapfs.NewMemMapFs(time.Now))
	if err := ExtractTarToVFS(lfsMemfs, tar.NewReader(gr)); err != nil {
		return nil, err
	}

	return lfsMemfs, nil
}

func ExtractTarToVFS(vfs VFS, t *tar.Reader) error {
	for {
		hdr, err := t.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		}

		if err := (func() error {
			hdr.Name = "/" + strings.TrimPrefix(strings.TrimSuffix(hdr.Name, "/"), "/")

			// Make parents
			vfs.MkdirAll(path.Dir(hdr.Name), 0777)

			mode := hdr.FileInfo().Mode()
			switch {
			case mode&fs.ModeDir > 0:
				err := vfs.Mkdir(hdr.Name, mode)
				switch {
				case os.IsExist(err):
					// Do nothing
				case err != nil:
					return err
				}
			case mode&fs.ModeSymlink > 0:
				if linker, ok := vfs.(afero.Linker); ok {
					if err := linker.SymlinkIfPossible(hdr.Linkname, hdr.Name); err != nil {
						switch {
						case os.IsExist(err):
							// Do nothing
						case err != nil:
							return err
						}
					}
				} else {
					return afero.ErrNoSymlink
				}
			default:
				fd, err := vfs.OpenFile(hdr.Name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
				if err != nil {
					return err
				}
				// Don't defer the close because it'll update the modification time.
				if _, err := io.CopyN(fd, t, hdr.Size); err != nil {
					fd.Close()
					return err
				}
				fd.Close()
			}

			if err := vfs.Chown(hdr.Name, hdr.Uid, hdr.Gid); err != nil {
				return err
			}
			if err := vfs.Chmod(hdr.Name, mode); err != nil {
				return err
			}
			if err := vfs.Chtimes(hdr.Name, hdr.FileInfo().ModTime(), hdr.FileInfo().ModTime()); err != nil {
				return err
			}

			return nil
		}()); err != nil {
			return fmt.Errorf("extracting %q: %v", hdr.Name, err)
		}
	}
}

func NewMemCopyOnWriteFs(base VFS, timeSource TimeSource) VFS {
	lfsMemfs := NewLinkingFs(memmapfs.NewMemMapFs(timeSource))
	return cowfs.NewCopyOnWriteFs(base, lfsMemfs)
}

func NewSymlinkResolvingRelativeFs(base VFS, Getwd func() (dir string)) VFS {
	rpos := &realpathOs{Getwd, base}
	return NewPathMappingFs(base, func(op FsOp, name string) (string, error) {
		switch op {
		case FsOpMkdir, FsOpCreate:
			dir, err := realpath.Realpath(rpos, path.Dir(name))
			// Expect at least one not exist, but we'll go as far as possible.
			if err != nil && errors.Is(err, fs.ErrNotExist) {
				return dir, err
			}
			return path.Join(dir, path.Base(name)), nil
		default:
			return realpath.Realpath(rpos, name)
		}
	})
}

type realpathOs struct {
	getwd func() (dir string)
	base  VFS
}

var _ realpath.OS = (*realpathOs)(nil)

func (r *realpathOs) Getwd() string {
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

// VFSToFS converts a VFS to Go's FS type.
func VFSToFS(vfs VFS) *vfsAdapter {
	return &vfsAdapter{vfs}
}

type vfsAdapter struct {
	vfs VFS
}

var _ fs.FS = (*vfsAdapter)(nil)
var _ WASMFS = (*vfsAdapter)(nil)

func (v *vfsAdapter) Open(name string) (fs.File, error) {
	af, err := v.vfs.Open(name)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fmt.Println("open")

	return af, nil
}

// String should return a human-readable format of the filesystem:
//   - If read-only, $host:$guestDir:ro
//   - If read-write, $host:$guestDir
//
// For example, if this filesystem is backed by the real directory
// "/tmp/wasm" and the GuestDir is "/", the expected value is
// "/var/tmp:/tmp".
//
// When the host filesystem isn't a real filesystem, substitute a symbolic,
// human-readable name. e.g. "virtual:/"
func (v *vfsAdapter) String() string {
	return "/:/"
}

// GuestDir is the name of the path the guest should use this filesystem
// for, or root ("/") for any files.
//
// This value allows the guest to avoid making file-system calls when they
// won't succeed. e.g. if "/tmp" is returned and the guest requests
// "/etc/passwd". This approach is used in compilers that use WASI
// pre-opens.
//
// # Notes
//   - Implementations must always return the same value.
//   - Go compiled with runtime.GOOS=js do not pay attention to this value.
//     Hence, you need to normalize the filesystem with NewRootFS to ensure
//     paths requested resolve as expected.
//   - Working directories are typically tracked in wasm, though possible
//     some relative paths are requested. For example, TinyGo may attempt
//     to resolve a path "../.." in unit tests.
//   - Zig uses the first path name it sees as the initial working
//     directory of the process.
func (v *vfsAdapter) GuestDir() string {
	return "/"
}

// Path is the name of the path the guest should use this filesystem for,
// or root ("/") if unknown.
//
// This value allows the guest to avoid making file-system calls when they
// won't succeed. e.g. if "/tmp" is returned and the guest requests
// "/etc/passwd". This approach is used in compilers that use WASI
// pre-opens.
//
// # Notes
//   - Go compiled with runtime.GOOS=js do not pay attention to this value.
//     Hence, you need to normalize the filesystem with NewRootFS to ensure
//     paths requested resolve as expected.
//   - Working directories are typically tracked in wasm, though possible
//     some relative paths are requested. For example, TinyGo may attempt
//     to resolve a path "../.." in unit tests.
//   - Zig uses the first path name it sees as the initial working
//     directory of the process.
func (v *vfsAdapter) Path() string {
	return "/"
}

// OpenFile is similar to os.OpenFile, except the path is relative to this
// file system.
//
// # Constraints on the returned file
//
// Implementations that can read flags should enforce them regardless of
// the type returned. For example, while os.File implements io.Writer,
// attempts to write to a directory or a file opened with os.O_RDONLY fail
// with an os.PathError of syscall.EBADF.
//
// Some implementations choose whether to enforce read-only opens, namely
// fs.FS. While fs.FS is supported (Adapt), wazero cannot runtime enforce
// open flags. Instead, we encourage good behavior and test our built-in
// implementations.
func (v *vfsAdapter) OpenFile(path string, flag int, perm fs.FileMode) (fs.File, error) {
	fmt.Printf("OpenFile: %q, flags %v, perm %v\n", path, flag, perm)
	fd, err := v.vfs.OpenFile(path, flag, perm)
	fmt.Printf("fd: %#v, err: %v\n", fd, err)

	return fd, err
}

// ^^ TODO: Consider syscall.Open, though this implies defining and
// coercing flags and perms similar to what is done in os.OpenFile.

// Mkdir is similar to os.Mkdir, except the path is relative to this file
// system.
func (v *vfsAdapter) Mkdir(path string, perm fs.FileMode) error {
	fmt.Println("Mkdir")

	return v.vfs.Mkdir(path, perm)
}

// ^^ TODO: Consider syscall.Mkdir, though this implies defining and
// coercing flags and perms similar to what is done in os.Mkdir.

// Rename is similar to syscall.Rename, except the path is relative to this
// file system.
//
// # Errors
//
// The following errors are expected:
//   - syscall.EINVAL: `from` or `to` is invalid.
//   - syscall.ENOENT: `from` or `to` don't exist.
//   - syscall.ENOTDIR: `from` is a directory and `to` exists, but is a file.
//   - syscall.EISDIR: `from` is a file and `to` exists, but is a directory.
//
// # Notes
//
//   -  Windows doesn't let you overwrite an existing directory.
func (v *vfsAdapter) Rename(from, to string) error {
	return v.vfs.Rename(from, to)
}

// Rmdir is similar to syscall.Rmdir, except the path is relative to this
// file system.
//
// # Errors
//
// The following errors are expected:
//   - syscall.EINVAL: `path` is invalid.
//   - syscall.ENOENT: `path` doesn't exist.
//   - syscall.ENOTDIR: `path` exists, but isn't a directory.
//   - syscall.ENOTEMPTY: `path` exists, but isn't empty.
//
// # Notes
//
//   - As of Go 1.19, Windows maps syscall.ENOTDIR to syscall.ENOENT.
func (v *vfsAdapter) Rmdir(path string) error {
	fmt.Println("Rmdir")

	// TODO: It's unclear whether this works or not
	return v.vfs.Remove(path)
}

// Unlink is similar to syscall.Unlink, except the path is relative to this
// file system.
//
// The following errors are expected:
//   - syscall.EINVAL: `path` is invalid.
//   - syscall.ENOENT: `path` doesn't exist.
//   - syscall.EISDIR: `path` exists, but is a directory.
func (v *vfsAdapter) Unlink(path string) error {
	fmt.Println("Unlink")

	// TODO: It's unclear whether this works or not
	return v.vfs.Remove(path)
}

// Utimes is similar to syscall.UtimesNano, except the path is relative to
// this file system.
//
// # Errors
//
// The following errors are expected:
//   - syscall.EINVAL: `path` is invalid.
//   - syscall.ENOENT: `path` doesn't exist
//
// # Notes
//
//   - To set wall clock time, retrieve it first from sys.Walltime.
//   - syscall.UtimesNano cannot change the ctime. Also, neither WASI nor
//     runtime.GOOS=js support changing it. Hence, ctime it is absent here.
func (v *vfsAdapter) Utimes(path string, atimeNsec, mtimeNsec int64) error {
	fmt.Println("Utimes")

	return v.vfs.Chtimes(path, time.Unix(0, atimeNsec), time.Unix(0, mtimeNsec))
}

type WASMFS interface {
	// String should return a human-readable format of the filesystem:
	//   - If read-only, $host:$guestDir:ro
	//   - If read-write, $host:$guestDir
	//
	// For example, if this filesystem is backed by the real directory
	// "/tmp/wasm" and the GuestDir is "/", the expected value is
	// "/var/tmp:/tmp".
	//
	// When the host filesystem isn't a real filesystem, substitute a symbolic,
	// human-readable name. e.g. "virtual:/"
	String() string

	// GuestDir is the name of the path the guest should use this filesystem
	// for, or root ("/") for any files.
	//
	// This value allows the guest to avoid making file-system calls when they
	// won't succeed. e.g. if "/tmp" is returned and the guest requests
	// "/etc/passwd". This approach is used in compilers that use WASI
	// pre-opens.
	//
	// # Notes
	//   - Implementations must always return the same value.
	//   - Go compiled with runtime.GOOS=js do not pay attention to this value.
	//     Hence, you need to normalize the filesystem with NewRootFS to ensure
	//     paths requested resolve as expected.
	//   - Working directories are typically tracked in wasm, though possible
	//     some relative paths are requested. For example, TinyGo may attempt
	//     to resolve a path "../.." in unit tests.
	//   - Zig uses the first path name it sees as the initial working
	//     directory of the process.
	GuestDir() string

	// Open is only defined to match the signature of fs.FS until we remove it.
	// Once we are done bridging, we will remove this function. Meanwhile,
	// using it will panic to ensure internal code doesn't depend on it.
	Open(name string) (fs.File, error)

	// OpenFile is similar to os.OpenFile, except the path is relative to this
	// file system, and syscall.Errno are returned instead of a os.PathError.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` or `flag` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist and `flag` doesn't contain
	//     os.O_CREATE.
	//
	// # Constraints on the returned file
	//
	// Implementations that can read flags should enforce them regardless of
	// the type returned. For example, while os.File implements io.Writer,
	// attempts to write to a directory or a file opened with os.O_RDONLY fail
	// with a syscall.EBADF.
	//
	// Some implementations choose whether to enforce read-only opens, namely
	// fs.FS. While fs.FS is supported (Adapt), wazero cannot runtime enforce
	// open flags. Instead, we encourage good behavior and test our built-in
	// implementations.
	OpenFile(path string, flag int, perm fs.FileMode) (fs.File, error)
	// ^^ TODO: Consider syscall.Open, though this implies defining and
	// coercing flags and perms similar to what is done in os.OpenFile.

	// Mkdir is similar to os.Mkdir, except the path is relative to this file
	// system, and syscall.Errno are returned instead of a os.PathError.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.EEXIST: `path` exists and is a directory.
	//   - syscall.ENOTDIR: `path` exists and is a file.
	//
	Mkdir(path string, perm fs.FileMode) error
	// ^^ TODO: Consider syscall.Mkdir, though this implies defining and
	// coercing flags and perms similar to what is done in os.Mkdir.

	// Rename is similar to syscall.Rename, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `from` or `to` is invalid.
	//   - syscall.ENOENT: `from` or `to` don't exist.
	//   - syscall.ENOTDIR: `from` is a directory and `to` exists, but is a file.
	//   - syscall.EISDIR: `from` is a file and `to` exists, but is a directory.
	//
	// # Notes
	//
	//   -  Windows doesn't let you overwrite an existing directory.
	Rename(from, to string) error

	// Rmdir is similar to syscall.Rmdir, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.ENOTDIR: `path` exists, but isn't a directory.
	//   - syscall.ENOTEMPTY: `path` exists, but isn't empty.
	//
	// # Notes
	//
	//   - As of Go 1.19, Windows maps syscall.ENOTDIR to syscall.ENOENT.
	Rmdir(path string) error

	// Unlink is similar to syscall.Unlink, except the path is relative to this
	// file system.
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.EISDIR: `path` exists, but is a directory.
	Unlink(path string) error

	// Utimes is similar to syscall.UtimesNano, except the path is relative to
	// this file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist
	//
	// # Notes
	//
	//   - To set wall clock time, retrieve it first from sys.Walltime.
	//   - syscall.UtimesNano cannot change the ctime. Also, neither WASI nor
	//     runtime.GOOS=js support changing it. Hence, ctime it is absent here.
	Utimes(path string, atimeNsec, mtimeNsec int64) error
}
