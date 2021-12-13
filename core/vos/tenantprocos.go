package vos

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"

	"josephlewis.net/osshit/core/logger"
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
	// Exec is the process executable that is run when the process starts.
	Exec ProcessFunc
}

var _ VOS = (*TenantProcOS)(nil)

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
func (ea *TenantProcOS) Getwd() (dir string) {
	return ea.Dir
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

func (ea *TenantProcOS) Run() (resultCode int) {
	defer func() {
		if r := recover(); r != nil {
			// Log the panic
			ea.TenantOS.eventRecorder.Record(&logger.LogEntry_Panic{
				Panic: &logger.Panic{
					Context:    fmt.Sprintf("Running %q got panic: %v", ea.ExecutablePath, r),
					Stacktrace: string(debug.Stack()),
				},
			})

			// Make it look like a crash to the user.
			fmt.Fprintf(ea.Stderr(), "%s: Segmentation fault\n", ea.ExecutablePath)
			resultCode = 2
		}
	}()

	if ea.Exec == nil {
		return 1
	}
	return ea.Exec(ea)
}

type ProcAttr struct {
	// If Dir is non-empty, the child changes into the directory before
	// creating the process.
	Dir string
	// If Env is non-empty, it gives the environment variables for the
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
	if len(attr.Env) == 0 {
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

	// Set up exec
	shellCmd, shellPath, shellErr := ea.findHoneypotCommand(out.ExecutablePath)
	execFsPath, execFsErr := LookPath(ea, out.ExecutablePath)

	switch {
	case shellErr == nil && execFsErr == nil:
		// Command found everywhere.
		out.Exec = shellCmd
		out.ExecutablePath = execFsPath

	case shellErr == nil && errors.Is(execFsErr, ErrNotFound):
		// Honeypot command found, but FS didn't have it. Run command anyway.
		out.Exec = shellCmd
		out.ExecutablePath = shellPath
	case errors.Is(shellErr, ErrNotFound) && execFsErr == nil:
		// The FS found the path but the honeypot didn't, fake a segfault
		out.Exec = segfault
		out.ExecutablePath = execFsPath

		ea.TenantOS.eventRecorder.Record(&logger.LogEntry_UnknownCommand{
			UnknownCommand: &logger.UnknownCommand{
				Command: ea.Args(),
				Status:  logger.UnknownCommand_NOT_IMPLEMENTED,
			},
		})
	case errors.Is(execFsErr, ErrNotFound):
		ea.TenantOS.eventRecorder.Record(&logger.LogEntry_UnknownCommand{
			UnknownCommand: &logger.UnknownCommand{
				Command: argv,
				Status:  logger.UnknownCommand_NOT_FOUND,
			},
		})
		return nil, fmt.Errorf("%s: command not found", out.ExecutablePath)
	default:
		ea.TenantOS.eventRecorder.Record(&logger.LogEntry_UnknownCommand{
			UnknownCommand: &logger.UnknownCommand{
				Command:      argv,
				ErrorMessage: fmt.Sprintf("honeypot err: %v FS err: %v", shellErr, execFsErr),
			},
		})
		return nil, fmt.Errorf("%s: permission denied", out.ExecutablePath)
	}

	ea.TenantOS.eventRecorder.Record(&logger.LogEntry_RunCommand{
		RunCommand: &logger.RunCommand{
			Command:              argv,
			EnvironmentVariables: env.Environ(),
			ResolvedCommandPath:  out.ExecutablePath,
		},
	})

	return out, nil
}

func (ea *TenantProcOS) LogInvalidInvocation(err error) {
	ea.TenantOS.eventRecorder.Record(&logger.LogEntry_InvalidInvocation{
		InvalidInvocation: &logger.InvalidInvocation{
			Command: ea.Args(),
			Error:   err.Error(),
		},
	})
}

func (ea *TenantProcOS) findHoneypotCommand(execPath string) (ProcessFunc, string, error) {
	switch {
	case !strings.Contains(execPath, "/"):
		// Not a fully qualified command path try under all $PATHs.
		for _, searchPath := range filepath.SplitList(ea.Getenv("PATH")) {
			if cmd, resPath, err := ea.findHoneypotCommand(path.Join(searchPath, execPath)); err == nil {
				return cmd, resPath, nil
			}
		}
		return nil, "", ErrNotFound

	case !path.IsAbs(execPath):
		// Not an absolute path, try again based on PWD
		execPath = path.Clean(path.Join(ea.Dir, execPath))
		fallthrough

	default:
		cmd := ea.TenantOS.sharedOS.processResolver(execPath)
		if cmd == nil {
			return nil, "", ErrNotFound
		}
		return cmd, execPath, nil
	}
}

func segfault(virtOS VOS) int {
	name := virtOS.Args()[0]
	fmt.Fprintf(virtOS.Stdout(), "%s: Segmentation fault\n", name)

	return 1
}

var _ ProcessFunc = segfault
