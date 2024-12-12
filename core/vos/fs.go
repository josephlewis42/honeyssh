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
