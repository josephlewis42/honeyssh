package core

import (
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotFound is the error resulting if a path search failed to find an executable file.
var ErrNotFound = exec.ErrNotFound

func findExecutable(vos VOS, file string) error {
	d, err := vos.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return fs.ErrPermission
}

// LookPath searches for an executable named file in the directories named by
// the PATH environment variable. If file contains a slash, it is tried directly
// and the PATH is not consulted. The result may be an absolute path or a path
// relative to the current directory.
func LookPath(vos VOS, file string) (string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(vos, file)
		if err == nil {
			return file, nil
		}
		return "", err
	}
	path := vos.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(vos, path); err == nil {
			return path, nil
		}
	}
	return "", ErrNotFound
}
