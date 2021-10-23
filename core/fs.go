package core

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	containerregistry "github.com/google/go-containerregistry/pkg/v1"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"josephlewis.net/osshit/third_party/realpath"
)

// WhiteoutPrefix prefix means file is a whiteout.
const WhiteoutPrefix = ".wh."

// VFS implements a virtual filesystem.
type VFS = afero.Fs

func WalkImgFs(layers []containerregistry.Layer, w io.Writer) error {
	whiteouts := make(map[string]bool)

	tw := tar.NewWriter(w)
	defer tw.Close()

	for layerIdx, layer := range layers {
		ul, err := layer.Uncompressed()
		if err != nil {
			return fmt.Errorf("couldn't decompress layer[%d]: %v", layerIdx, err)
		}
		defer ul.Close()

		tarReader := tar.NewReader(ul)
		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break // End of archive
			}
			if err != nil {
				return fmt.Errorf("couldn't read next file in layer[%d]: %v", layerIdx, err)
			}

			if strings.HasPrefix(path.Base(hdr.FileInfo().Name()), WhiteoutPrefix) {
				whiteouts[hdr.Name] = true
			}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if hdr.FileInfo().Size() > 0 {
				if _, err := io.Copy(tw, tarReader); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NewCopyOnWriteFs(tarReader *tar.Reader) VFS {
	base := tarfs.New(tarReader)
	roBase := NewLoggingFs(afero.NewReadOnlyFs(base), "tar-ro")
	lfsRoBase := NewLoggingFs(NewLinkingFs(roBase), "ln-tar-ro")

	memFs := NewLoggingFs(afero.NewMemMapFs(), "mem")
	lfsMemfs := NewLoggingFs(&LinkingFsWrapper{memFs}, "ln-mem")

	ufs := afero.NewCopyOnWriteFs(lfsRoBase, lfsMemfs)

	NewLoggingFs(ufs, "union")
	return lfsRoBase
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
	// if lstater, ok := lfs.VFS.(afero.Lstater); ok {
	// 	return lstater.LstatIfPossible(name)
	// }

	fi, err := lfs.VFS.Stat(name)
	return fi, true, err
}

func (lfs *LinkingFsWrapper) ReadlinkIfPossible(name string) (string, error) {
	// if linker, ok := lfs.VFS.(afero.LinkReader); ok {
	// 	return linker.ReadlinkIfPossible(name)
	// }

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
	// if linker, ok := lfs.VFS.(afero.Linker); ok {
	// 	return linker.SymlinkIfPossible(oldname, newname)
	// }

	return afero.WriteFile(lfs.VFS, newname, ([]byte)(oldname), 0666|os.ModeSymlink)
}
