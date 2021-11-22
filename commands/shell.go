package commands

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/abiosoft/readline"
	"github.com/anmitsu/go-shlex"
	"josephlewis.net/osshit/core/vos"
)

const (
	EnvHome     = "HOME"
	EnvPWD      = "PWD"
	EnvPath     = "PATH"
	EnvPrompt   = "PS1"
	EnvHostname = "HOSTNAME"
	EnvUser     = "USER"
	EnvUID      = "UID"

	DefaultPrompt = `\u@\h:\w\$ `
)

var (
	envRegex = regexp.MustCompile(`(\$\$|\$\w+)`)
)

type Shell struct {
	VirtualOS vos.VOS
	Readline  *readline.Instance
	lastRet   int
	history   []string

	// Set to true to quit the shell
	Quit bool
}

func RunShell(virtualOS vos.VOS) int {
	s, err := NewShell(virtualOS)
	if err != nil {
		fmt.Fprintf(virtualOS.Stderr(), "sh: %s\n", err)
		return 1
	}
	s.Run()
	return 0
}

func NewShell(virtualOS vos.VOS) (*Shell, error) {

	cfg := &readline.Config{
		Stdin:  readline.NewCancelableStdin(virtualOS.Stdin()),
		Stdout: virtualOS.Stdout(),
		Stderr: virtualOS.Stderr(),
		FuncGetWidth: func() int {
			return virtualOS.GetPTY().Width
		},
		FuncIsTerminal: func() bool {
			return virtualOS.GetPTY().IsPTY
		},
	}

	if err := cfg.Init(); err != nil {
		return nil, err
	}

	readline, err := readline.NewEx(cfg)
	if err != nil {
		return nil, err
	}

	shell := &Shell{
		VirtualOS: virtualOS,
		Readline:  readline,
	}

	shell.Init(virtualOS.SSHUser())

	return shell, nil
}

// Init sets up the environment similar to login + source ~/.bashrc.
func (s *Shell) Init(username string) {
	var homedir string
	if s.VirtualOS.Getuid() == 0 {
		homedir = "/root"
	} else {
		homedir = fmt.Sprintf("/home/%s", username)
	}

	s.VirtualOS.Setenv(EnvHome, homedir)
	// Use chdir in case the dir doesn't exist.
	_ = s.VirtualOS.Chdir(homedir)

	s.VirtualOS.Setenv(EnvPath, "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	host, _ := s.VirtualOS.Hostname()
	s.VirtualOS.Setenv(EnvHostname, host)
	s.VirtualOS.Setenv(EnvPrompt, DefaultPrompt)
	if wd, err := s.VirtualOS.Getwd(); err != nil {
		s.VirtualOS.Setenv(EnvPWD, wd)
	}
	s.VirtualOS.Setenv(EnvUser, username)
	s.VirtualOS.Setenv(EnvUID, fmt.Sprintf("%d", s.VirtualOS.Getuid()))
}

func (s *Shell) Prompt() string {
	prompt := s.VirtualOS.Getenv(EnvPrompt)
	if prompt == "" {
		prompt = DefaultPrompt
	}
	prompt = strings.ReplaceAll(prompt, `\u`, s.VirtualOS.Getenv(EnvUser))
	prompt = strings.ReplaceAll(prompt, `\h`, s.VirtualOS.Getenv(EnvHostname))

	pwd, _ := s.VirtualOS.Getwd()
	home, _ := s.VirtualOS.UserHomeDir()
	if strings.HasPrefix(pwd, home) {
		pwd = "~" + strings.TrimPrefix(pwd, home)
	}

	prompt = strings.ReplaceAll(prompt, `\w`, pwd)

	if s.VirtualOS.Getuid() == 0 {
		prompt = strings.ReplaceAll(prompt, `\$`, "#")
	} else {
		prompt = strings.ReplaceAll(prompt, `\$`, "$")
	}

	return prompt
}

func (s *Shell) Run() {
	for !s.Quit {
		s.Readline.SetPrompt(s.Prompt())
		line, err := s.Readline.Readline()

		// This doesn't make sense for shell, but it needs to be kept in line with
		// the readline history.
		s.history = append(s.history, line)

		switch {
		case err == io.EOF:
			return // Input closed, quit.

		case err == readline.ErrInterrupt:
			// TODO: handle interrupt, line is valid here.
			log.Printf("interrupt")

		case err != nil:
			log.Printf("Error readline: %v", err)
			continue

		case len(line) == 0:
			continue // empty line

		default:
			tokens, err := shlex.Split(line, true)
			if err != nil {
				fmt.Fprintln(s.Readline, "-bash: syntax error: unexpected end of file")
				continue
			}
			if len(tokens) == 0 {
				continue
			}

			// Take off command environment variables
			effectiveEnv := s.VirtualOS.Environ()
			var cmdEnvStop int
			for ; cmdEnvStop < len(tokens); cmdEnvStop++ {
				tok := tokens[cmdEnvStop]
				if strings.Contains(tok, "=") {
					effectiveEnv = append(effectiveEnv, tok)
				} else {
					break
				}
			}

			tokens = tokens[cmdEnvStop:]

			// If the full command was environment variables, set them. Otherwise they
			// should only be populated for the upcoming command.
			if 0 == len(tokens) {
				vos.CopyEnv(s.VirtualOS, vos.NewMapEnvFromEnvList(effectiveEnv))
				continue
			}

			// Expand the environment
			for i, tok := range tokens {
				mapEnv := vos.NewMapEnvFromEnvList(effectiveEnv)

				// Shell only arguments
				mapEnv.Setenv("$", fmt.Sprintf("%d", s.VirtualOS.Getpid()))
				mapEnv.Setenv("?", fmt.Sprintf("%d", uint8(s.lastRet)))
				mapEnv.Setenv("WIDTH", fmt.Sprintf("%d", s.VirtualOS.GetPTY().Width))
				mapEnv.Setenv("HEIGHT", fmt.Sprintf("%d", s.VirtualOS.GetPTY().Height))

				tokens[i] = os.Expand(tok, mapEnv.Getenv)
			}

			s.ExecuteProgramOrBuiltin(effectiveEnv, tokens)
		}
	}
}

func (s *Shell) ExecuteProgramOrBuiltin(cmdEnv []string, args []string) {
	// Execute builtins
	if builtin, ok := AllBuiltins[args[0]]; ok {
		s.lastRet = builtin.Main(s, args)
		return
	}

	s.ExecuteProgram(cmdEnv, args)
}

func (s *Shell) ExecuteProgram(cmdEnv []string, args []string) {

	shellCmd, shellPath, shellErr := FindCommand(s.VirtualOS, args[0])
	execFsPath, execFsErr := vos.LookPath(s.VirtualOS, args[0])

	switch {
	case shellErr == nil && execFsErr == vos.ErrNotFound:
		// Do nothing, always execute a found honeypot command, even if the FS says
		// it doesn't exist.
		execFsPath = shellPath
	case shellErr == vos.ErrNotFound && execFsErr == nil:
		// The FS found the path but the honeypot didn't.
		shellCmd = SegfaultCommand
	case shellErr == vos.ErrNotFound || execFsErr == vos.ErrNotFound:
		fmt.Fprintf(s.Readline, "%s: command not found\n", args[0])
		return
	case shellErr != nil || execFsErr != nil:
		fmt.Fprintf(s.Readline, "%s: permission denied\n", args[0])
		return
	}

	// TODO log execution
	proc, err := s.VirtualOS.StartProcess(execFsPath, args, &vos.ProcAttr{
		Env:   cmdEnv,
		Files: s.VirtualOS,
	})
	if err != nil {
		fmt.Fprintf(s.Readline, "%s: %s\n", args[0], err)
	}

	s.lastRet = shellCmd.Main(proc)
}

func FindCommand(virtualOS vos.VOS, execPath string) (HoneypotCommand, string, error) {
	switch {
	case !strings.Contains(execPath, "/"):
		// Not a fully qualified command path try under all $PATHs.
		for _, searchPath := range filepath.SplitList(virtualOS.Getenv("PATH")) {
			if cmd, resPath, err := FindCommand(virtualOS, path.Join(searchPath, execPath)); err == nil {
				return cmd, resPath, nil
			}
		}
		return nil, "", vos.ErrNotFound

	case !path.IsAbs(execPath):
		// Not an absolute path, try again., try again.
		wd, err := virtualOS.Getwd()
		if err != nil {
			return nil, "", err
		}
		execPath = path.Clean(path.Join(wd, execPath))
		fallthrough

	default:
		cmd, ok := AllCommands[execPath]
		if !ok {
			return nil, "", vos.ErrNotFound
		}
		return cmd, execPath, nil
	}
}

func init() {
	addBinCmd("sh", HoneypotCommandFunc(RunShell))
}
