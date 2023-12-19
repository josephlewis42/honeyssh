package vfs

import (
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tetratelabs/wazero/experimental/sys"
)

type Inode = int64

type Memory struct {
	inodes   map[Inode]*inodeData
	inodeCtr atomic.Int64

	Now func() time.Time
}

func NewMemory() *Memory {
	return &Memory{
		inodes: map[Inode]*inodeData{
			0: {
				mode:     0755,
				uid:      0,
				gid:      0,
				mtime:    0,
				ctime:    0,
				nlink:    0,
				children: map[string]Inode{},
			},
		},
		Now: time.Now,
	}
}

func (fs *Memory) insertInode(inode *inodeData) Inode {
	newInode := fs.inodeCtr.Add(1)
	fs.inodes[newInode] = inode
	inode.inode = newInode
	return newInode
}

type VFSContext struct {
	UserID   int
	GroupIDs []int
	PWD      string
	Umask    uint32
}

type inodeResult struct {
	parent *inodeData
	inode  *inodeData
}

func (fs *Memory) FindInode(fsContext VFSContext, file string) (*inodeResult, Errno) {

	// https://man7.org/linux/man-pages/man7/path_resolution.7.html

	// resolveRelative contains the path to which relative links will be resolved.
	resolveRelative := fsContext.PWD

	// number of symlinks allowed to resolve before erroring.
	symlinksAllowed := 40 + 1

resolve:
	symlinksAllowed--
	if symlinksAllowed <= 0 {
		return nil, sys.ELOOP
	}

	// make the path absolute.
	if !path.IsAbs(file) {
		file = path.Join(resolveRelative, file)
	}

	if file == "" {
		return nil, sys.ENOENT
	}

	// split to consumable parts
	pathParts := strings.Split(file, "/")
	// Inode at the current path. Root is always a directory.
	currentInode := fs.inodes[0]
	resolveRelative = "/"

	for len(pathParts) > 1 {
		pathName := pathParts[0]
		pathParts = pathParts[1:]

		resolveRelative += pathName + "/"

		nextInode, errno := currentInode.getChild(pathName)
		if errno != 0 {
			return nil, errno
		}
		currentInode = fs.inodes[nextInode]

		switch currentInode.inodeKind {
		default:
			return nil, sys.ENOTDIR
		case kindSymlink:
			// TODO: check permissions
			file = path.Join(currentInode.destination, path.Join(pathParts...))
			goto resolve
		case kindDirectory:
			// TODO: check permissions
			continue
		}
	}

	// Last inode
	result := inodeResult{
		parent: currentInode,
	}

	currentInode.getChild(pathName)

	// TODO: check permissions
	return &result, 0
}

func (fs *Memory) FindLinkInode(fsContext VFSContext, file string) (*inodeData, Errno) {
	base, linkName := path.Split(file)
	inode, errno := fs.FindInode(fsContext, base)
	if errno != 0 {
		return nil, errno
	}

	nodeNum, errno := inode.getChild(linkName)
	if errno != 0 {
		return nil, errno
	}

	return fs.inodes[nodeNum].assertSymlink()
}

func (fs *Memory) FindDirectoryInode(fsContext VFSContext, file string) (*inodeData, Errno) {
	inode, errno := fs.FindInode(fsContext, file)
	if errno != 0 {
		return nil, errno
	}
	return inode.assertDirectory()
}

// https://pubs.opengroup.org/onlinepubs/009695299/functions/mkdir.html
// https://pubs.opengroup.org/onlinepubs/009695299/functions/open.html
func (fs *Memory) createChildInode(fsContext VFSContext, file string, perm fs.FileMode) (*inodeData, Errno) {
	base, dirName := path.Split(file)
	parentInode, errno := fs.FindDirectoryInode(fsContext, base)
	if errno != 0 {
		return nil, errno
	}

	if errno := parentInode.assertChildNotExist(dirName); errno != 0 {
		return nil, errno
	}

	now := fs.Now().UnixNano()

	newInode := &inodeData{
		mode:  uint32(perm) & ^fsContext.Umask & 0777,
		uid:   fsContext.UserID,
		gid:   parentInode.gid,
		mtime: now,
		ctime: now,
		nlink: 1, // Parent link
	}

	parentInode.children[dirName] = fs.insertInode(newInode)

	return newInode, Success
}

// https://pubs.opengroup.org/onlinepubs/009695299/functions/mkdir.html
func (fs *Memory) Mkdir(fsContext VFSContext, file string, perm fs.FileMode) Errno {
	newInode, errno := fs.createChildInode(fsContext, file, perm)
	if errno != 0 {
		return errno
	}

	newInode.children = map[string]Inode{}

	return Success
}

// https://pubs.opengroup.org/onlinepubs/009695299/functions/open.html
func (fs *Memory) Open(fsContext VFSContext, name string, flags int, perm os.FileMode) (*inodeHandle, Errno) {
	// TODO: Validate that exactly one of O_RDONLY, O_WRONLY, or O_RDWR are specified.

	inode, err := fs.FindInode(fsContext, name)
	switch {
	case err == Success:
		// Check that EXCL not set to prevent clobber.
		if os.O_EXCL&flags > 0 {
			return nil, sys.EEXIST
		}

		return inode.newHandle(path.Base(name), flags), Success

	case err == sys.ENOENT && os.O_CREATE&flags > 0:
		inode, childErr := fs.createChildInode(fsContext, name, perm)
		if childErr != Success {
			return nil, childErr
		}

		return inode.newHandle(name, flags), Success

	default:
		return nil, err
	}
}

// https://pubs.opengroup.org/onlinepubs/009695299/functions/remove.html
func (fs *Memory) Remove(fsContext VFSContext, name string) Errno {

	return Success
}

// The rmdir() function shall remove a directory whose name is given by path. The directory shall be removed only if it is an empty directory.

// If the directory is the root directory or the current working directory of any process, it is unspecified whether the function succeeds, or whether it shall fail and set errno to [EBUSY].

// If path names a symbolic link, then rmdir() shall fail and set errno to [ENOTDIR].

// If the path argument refers to a path whose final component is either dot or dot-dot, rmdir() shall fail.

// If the directory's link count becomes 0 and no process has the directory open, the space occupied by the directory shall be freed and the directory shall no longer be accessible. If one or more processes have the directory open when the last link is removed, the dot and dot-dot entries, if present, shall be removed before rmdir() returns and no new entries may be created in the directory, but the directory shall not be removed until all references to the directory are closed.

// If the directory is not an empty directory, rmdir() shall fail and set errno to [EEXIST] or [ENOTEMPTY].

// Upon successful completion, the rmdir() function shall mark for update the st_ctime and st_mtime fields of the parent directory.

func (fs *Memory) Rmdir(fsContext VFSContext, name string) Errno {
	return Success
}

// The unlink() function shall remove a link to a file. If path names a symbolic link, unlink() shall remove the symbolic link named by path and shall not affect any file or directory named by the contents of the symbolic link. Otherwise, unlink() shall remove the link named by the pathname pointed to by path and shall decrement the link count of the file referenced by the link.
// When the file's link count becomes 0 and no process has the file open, the space occupied by the file shall be freed and the file shall no longer be accessible. If one or more processes have the file open when the last link is removed, the link shall be removed before unlink() returns, but the removal of the file contents shall be postponed until all references to the file are closed.
// The path argument shall not name a directory unless the process has appropriate privileges and the implementation supports using unlink() on directories.
// Upon successful completion, unlink() shall mark for update the st_ctime and st_mtime fields of the parent directory. Also, if the file's link count is not 0, the st_ctime field of the file shall be marked for update.
func (fs *Memory) Unlink(fsContext VFSContext, name string) Errno {
	return Success
}

type inodeDataKind int

const (
	kindNone inodeDataKind = iota
	kindFile
	kindDirectory
	kindSymlink
)

type inodeData struct {
	mutex sync.Mutex

	inode Inode

	mode  uint32 // Permission and mode bits
	uid   int    // User ID of owner.
	gid   int    // Group ID of owner.
	mtime int64  // Nanoseconds since the epoch of last modification.
	ctime int64  // Nanoseconds since the epoch of creation.
	nlink int    // Number of hard links incoming for this inode.

	inodeKind inodeDataKind

	// Only set if inodeKind is inodeKind is kindFile
	contents []byte

	// Only set if inodeKind is kindDirectory
	children map[string]Inode

	// Only set if inodeKind is kindSymlink
	destination string
}

type inodeHandle struct {
	*inodeData
	name     string
	readOnly bool
}

func (inode *inodeData) newHandle(name string, flags int) *inodeHandle {
	// TODO: Handle O_APPEND, O_TRUNC
	return &inodeHandle{
		inodeData: inode,
		name:      name,
		readOnly:  flags&os.O_RDONLY > 0,
	}
}

// IsDir implements fs.FileInfo
func (inode *inodeHandle) IsDir() bool {
	return inode.inodeKind == kindDirectory
}

// ModTime implements fs.FileInfo
func (inode *inodeHandle) ModTime() time.Time {
	return time.Unix(0, inode.mtime)
}

// Mode implements fs.FileInfo
func (inode *inodeHandle) Mode() fs.FileMode {
	// TODO: set directory and special bits
	return fs.FileMode(inode.mode)
}

// Mode implements fs.FileInfo
func (inode *inodeHandle) Name() string {
	return inode.name
}

// Size implements fs.FileInfo
func (inode *inodeHandle) Size() int64 {
	switch inode.inodeKind {
	case kindSymlink:
		return int64(len(inode.destination))
	case kindFile:
		return int64(len(inode.contents))
	default:
		return 0
	}
}

// Sys implements fs.FileInfo
func (inode *inodeHandle) Sys() any {
	return nil
}

var _ fs.FileInfo = (*inodeHandle)(nil)

// Truncate a file to a specific length.
//
// https://pubs.opengroup.org/onlinepubs/009695299/functions/ftruncate.html
func (inode *inodeHandle) Truncate(length int) Errno {
	if length < 0 || inode.readOnly {
		return sys.EINVAL
	}

	if _, err := inode.assertFile(); err != 0 {
		return err
	}

	switch {
	case len(inode.contents) > length:
		inode.contents = inode.contents[:length]
	case len(inode.contents) < length:
		inode.contents = append(inode.contents, make([]byte, length-len(inode.contents))...)
	}

	return Success
}

func (inode *inodeData) assertChildNotExist(name string) Errno {
	if _, err := inode.assertDirectory(); err != 0 {
		return err
	}

	if _, ok := inode.children[name]; ok {
		return sys.EEXIST
	}

	return 0
}

func (inode *inodeData) getChild(name string) (Inode, Errno) {
	if _, err := inode.assertDirectory(); err != 0 {
		return 0, sys.ENOTDIR
	}

	nextInode, ok := inode.children[name]
	if !ok {
		return 0, sys.ENOENT
	}

	return nextInode, 0
}

func (inode *inodeData) assertSymlink() (*inodeData, Errno) {
	if inode == nil || inode.inodeKind != kindSymlink {
		return nil, sys.EINVAL
	}

	return inode, 0
}

func (inode *inodeData) assertDirectory() (*inodeData, Errno) {
	if inode == nil || inode.inodeKind != kindDirectory {
		return nil, sys.ENOTDIR
	}

	return inode, 0
}

func (inode *inodeData) assertFile() (*inodeData, Errno) {
	if inode == nil || inode.inodeKind != kindFile {
		return nil, sys.EINVAL
	}

	return inode, 0
}
