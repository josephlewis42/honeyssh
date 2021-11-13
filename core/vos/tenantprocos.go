package vos

import (
	"fmt"
	"os"
	"path"
)

type TenantProcOS struct {
	*TenantOS

	VEnv

	VFS

	VIO

	// Path to the executable that started the process, errors if blank.
	ExecutablePath string
	// Args holds command line arguments, including the command as Args[0].
	ProcArgs []string
	// The process ID of the process
	PID int
	// The user ID of the process.
	UID int
	// Dir specifies the working directory of the command.
	Dir string
}

var _ VOS = (*TenantProcOS)(nil)

func (ea *TenantProcOS) Executable() (string, error) {
	if ea.ExecutablePath == "" {
		return "", os.ErrNotExist
	}

	return ea.ExecutablePath, nil
}

// Args implements VOS.Args.
func (ea *TenantProcOS) Args() []string {
	return ea.ProcArgs
}

// Getpid implements VOS.Getpid.
func (ea *TenantProcOS) Getpid() int {
	return ea.PID
}

// Getuid implements VOS.Getuid.
func (ea *TenantProcOS) Getuid() int {
	return ea.UID
}

// Getwd implements VOS.Getwd.
func (ea *TenantProcOS) Getwd() (dir string, err error) {
	return ea.Dir, nil
}

// Chdir implements VOS.Chdir.
func (ea *TenantProcOS) Chdir(dir string) (err error) {
	if !path.IsAbs(dir) {
		dir = path.Clean(path.Join(ea.Dir, dir))
	}

	stat, err := ea.Stat(dir)
	switch {
	case err != nil:
		return fmt.Errorf("%s: %v", dir, err)
	case !stat.IsDir():
		return fmt.Errorf("%s: Not a directory", dir)
	default:
		ea.Dir = dir
		return nil
	}
}

type ProcAttr struct {
	// If Dir is non-empty, the child changes into the directory before
	// creating the process.
	Dir string
	// If Env is non-nil, it gives the environment variables for the
	// new process in the form returned by Environ.
	// If it is nil, the result of Environ will be used.
	Env []string

	// Files specifies the open files inherited by the new process.
	Files VIO

	// Operating system-specific process creation attributes.
	// Note that setting this field means that your program
	// may not execute properly or even compile on some
	// operating systems.
	//Sys *syscall.SysProcAttr
}

// StartProcess starts a new process with the program, arguments and attributes
// specified by name, argv and attr. The argv slice will become os.Args in the
// new process, so it normally starts with the program name.
func (ea *TenantProcOS) StartProcess(name string, argv []string, attr *ProcAttr) (VOS, error) {
	if attr == nil {
		attr = &ProcAttr{}
	}

	if argv == nil {
		argv = []string{name}
	}

	var env VEnv
	if attr.Env == nil {
		env = NewMapEnvFrom(ea.VEnv)
	} else {
		env = NewMapEnvFromEnvList(attr.Env)
	}

	out := &TenantProcOS{
		TenantOS:       ea.TenantOS,
		VEnv:           env,
		ExecutablePath: name,
		ProcArgs:       argv,
		PID:            ea.TenantOS.sharedOS.NextPID(),
		UID:            ea.UID,
		Dir:            ea.Dir,
	}

	out.VFS = NewSymlinkResolvingRelativeFs(ea.TenantOS.fs, out.Getwd)

	if attr.Files == nil {
		out.VIO = NewNullIO()
	} else {
		out.VIO = attr.Files
	}

	if attr.Dir != "" {
		if err := out.Chdir(attr.Dir); err != nil {
			return nil, err
		}
	}

	return out, nil
}
