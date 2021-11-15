package core

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/abiosoft/readline"
	"github.com/anmitsu/go-shlex"
	"github.com/gliderlabs/ssh"
	"josephlewis.net/osshit/commands"
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
}

func NewShell(s ssh.Session, virtualOS vos.VOS) (*Shell, error) {

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

	shell.Init(s.User())

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
	for {
		s.Readline.SetPrompt(s.Prompt())
		line, err := s.Readline.Readline()

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
			for i, tok := range tokens {
				tokens[i] = os.Expand(tok, func(env string) string {
					switch {
					case env == "$": // $$
						return fmt.Sprintf("%d", s.VirtualOS.Getpid())
					case env == "?": // $?
						return fmt.Sprintf("%d", uint8(s.lastRet))
					case env == "WIDTH":
						return fmt.Sprintf("%d", s.VirtualOS.GetPTY().Width)
					case env == "HEIGHT":
						return fmt.Sprintf("%d", s.VirtualOS.GetPTY().Height)
					default:
						return s.VirtualOS.Getenv(env)
					}
				})
			}

			if len(tokens) == 0 {
				continue
			}

			// TODO check for = in paths to set up environment variables for commnad
			// to come.
			switch tokens[0] {
			case "exit":
				return
			case "cd":
				s.builtinCd(tokens)
				continue
			}

			execPath, err := vos.LookPath(s.VirtualOS, tokens[0])
			switch {
			case err == vos.ErrNotFound:
				fmt.Fprintf(s.Readline, "%s: command not found\n", tokens[0])
				continue
			case err == fs.ErrPermission || err != nil:
				fmt.Fprintf(s.Readline, "%s: permission denied\n", tokens[0])
				continue
			}

			if honeypotCommand, ok := commands.AllCommands[execPath]; ok {
				// TODO log execution

				proc, err := s.VirtualOS.StartProcess(execPath, tokens, &vos.ProcAttr{
					Env:   s.VirtualOS.Environ(),
					Files: s.VirtualOS,
				})
				if err != nil {
					fmt.Fprintf(s.Readline, "%s: %s\n", tokens[0], err)
					continue
				}

				s.lastRet = honeypotCommand.Main(proc)
			} else {
				fmt.Fprintf(s.Readline, "%s: command not found\n", tokens[0])
				continue
			}
		}
	}
}

func (s *Shell) builtinCd(args []string) {
	switch len(args) {
	case 1:
		args = append(args, s.VirtualOS.Getenv(EnvHome))
		fallthrough
	case 2:
		if err := s.VirtualOS.Chdir(args[1]); err != nil {
			fmt.Fprintf(s.VirtualOS.Stderr(), "%s: %v\n", args[0], err)
		}
	default:
		fmt.Fprintf(s.VirtualOS.Stderr(), "%s: too many arguments\n", args[0])
	}
}

// builtins
// pushd
// popd
// pwd
// cd
// exit

type listCloser []io.Closer

func (lc listCloser) Close() error {
	var lastErr error
	for _, v := range lc {
		if err := v.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}
