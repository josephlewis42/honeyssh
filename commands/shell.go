package commands

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/abiosoft/readline"
	"github.com/anmitsu/go-shlex"
	"josephlewis.net/osshit/core/vos"
)

const (
	EnvHome            = "HOME"
	EnvPWD             = "PWD"
	EnvPath            = "PATH"
	EnvPrompt          = "PS1"
	EnvHostname        = "HOSTNAME"
	EnvUser            = "USER"
	EnvUID             = "UID"
	DefaultColorPrompt = `\033[01;32m\u@\h\033[00m:\033[01;34m\w\033[00m\$ `
	DefaultPrompt      = `\u@\h:\w\$ `
)

var (
	envRegex = regexp.MustCompile(`(\$\$|\$\w+)`)
)

type Shell struct {
	VirtualOS vos.VOS
	Readline  *readline.Instance

	lastRet int
	history []string

	// Set to true to quit the shell
	Quit bool
}

func RunShell(virtualOS vos.VOS) int {

	s, err := NewShell(virtualOS)
	if err != nil {
		fmt.Fprintf(virtualOS.Stderr(), "sh: %s\n", err)
		return 1
	}

	cmd := &SimpleCommand{
		Use:       "sh [options] ...",
		Short:     "Standard command interpreter for the system. Currently being changed to conform with the POSIX 1003.2 standard.",
		NeverBail: true,
	}

	return cmd.Run(virtualOS, s.Run)
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
	host := s.VirtualOS.Hostname()
	s.VirtualOS.Setenv(EnvHostname, host)
	if s.VirtualOS.GetPTY().IsPTY {
		s.VirtualOS.Setenv(EnvPrompt, DefaultColorPrompt)
	} else {
		s.VirtualOS.Setenv(EnvPrompt, DefaultPrompt)
	}
	s.VirtualOS.Setenv(EnvPWD, s.VirtualOS.Getwd())
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

	pwd := s.VirtualOS.Getwd()
	home := s.VirtualOS.Getenv(EnvHome)
	if strings.HasPrefix(pwd, home) {
		pwd = "~" + strings.TrimPrefix(pwd, home)
	}

	prompt = strings.ReplaceAll(prompt, `\w`, pwd)

	if s.VirtualOS.Getuid() == 0 {
		prompt = strings.ReplaceAll(prompt, `\$`, "#")
	} else {
		prompt = strings.ReplaceAll(prompt, `\$`, "$")
	}

	return unescape(prompt)
}

func (s *Shell) Run() int {
	for !s.Quit {
		s.Readline.SetPrompt(s.Prompt())
		line, err := s.Readline.Readline()

		// This doesn't make sense for shell, but it needs to be kept in line with
		// the readline history.
		s.history = append(s.history, line)

		switch {
		case err == io.EOF:
			return 1 // Input closed, quit.

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
				vos.CopyEnv(s.VirtualOS, effectiveEnv)
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
	return 0
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
	proc, err := s.VirtualOS.StartProcess(args[0], args, &vos.ProcAttr{
		Env:   cmdEnv,
		Files: s.VirtualOS,
	})
	if err != nil {
		fmt.Fprintf(s.Readline, "sh: %s\n", err)
		return
	}

	s.lastRet = proc.Run()
}

func init() {
	addBinCmd("sh", RunShell)
}
