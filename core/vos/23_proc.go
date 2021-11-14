package vos

import (
	"io"
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

// func Exec(vos VOS, cmd string, args []string) Layer2VOS {
// 	pwd, _ := vos.Getwd()
// 	execPath, _ := LookPath(vos, cmd)
// 	return &FakeProc{
// 		Layer1VOS: vos,
// 		VIO:       vos,
// 		// TODO: make copy of env.
// 		VEnv: vos,
// 		MockProc: MockProc{
// 			ExecutablePath: execPath,
// 			ProcArgs:       append([]string{cmd}, args...),
// 			PID:            vos.Getpid() + 1,
// 			UID:            vos.Getuid(),
// 			Dir:            pwd,
// 		},
// 	}
// }

type VProc interface {
	// Returns the path to the executable that started the process.
	Executable() (string, error)

	// Getpid returns the process id of the caller.
	Getpid() int

	// Getuid returns the numeric user id of the caller.
	Getuid() int

	// Returns the arguments to the current process.
	Args() []string

	// Getwd returns a rooted path name corresponding to the current directory.
	Getwd() (dir string, err error)

	// Chdir changes the directory.
	Chdir(dir string) error
}

// Cmd is similar to go's os/exec.Cmd.
type Cmd struct {
	// Path is the path of the command to run.
	Path string

	// Args holds command line arguments, including the command as Args[0].
	// If the Args field is empty or nil, Run uses {Path}.
	//
	// In typical use, both Path and Args are set by calling Command.
	Args []string

	// Env specifies the environment of the process.
	// Each entry is of the form "key=value".
	// If Env is nil, the new process uses the current process's
	// environment.
	// If Env contains duplicate environment keys, only the last
	// value in the slice for each duplicate key is used.
	// As a special case on Windows, SYSTEMROOT is always added if
	// missing and not explicitly set to the empty string.
	Env []string

	// Dir specifies the working directory of the command.
	// If Dir is the empty string, Run runs the command in the
	// calling process's current directory.
	Dir string

	// Stdin specifies the process's standard input.
	Stdin io.Reader

	// Stdout and Stderr specify the process's standard output and error.
	Stdout io.Writer
	Stderr io.Writer
}