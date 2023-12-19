package vfs

import (
	wazerosys "github.com/tetratelabs/wazero/experimental/sys"
)

type FS = wazerosys.FS
type File = wazerosys.File

type UnimplementedFS = wazerosys.UnimplementedFS
type UnimplementedFile = wazerosys.UnimplementedFile

type Errno = wazerosys.Errno

const Success wazerosys.Errno = 0
