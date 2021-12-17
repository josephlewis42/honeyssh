package memmapfs

import "github.com/spf13/afero"

// Types to reduce the delta between this fork of memmapfs and afero's

const FilePathSeparator = "/"

type Fs = afero.Fs
type File = afero.File

var ErrFileExists = afero.ErrFileExists
var ErrFileNotFound = afero.ErrFileNotFound
