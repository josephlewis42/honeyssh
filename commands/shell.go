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
	"github.com/josephlewis42/honeyssh/core/vos"
	"mvdan.cc/sh/v3/syntax"
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
	commandFlag := cmd.Flags().String('c', "", "Command")

	return cmd.Run(virtualOS, func() int {
		if *commandFlag != "" {
			s.runCommand(*commandFlag)
			return s.lastRet
		}

		return s.runInteractive2()
	})
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

func (s *Shell) runInteractive2() int {
	// runner, err := interp.New(
	// 	interp.Dir(s.VirtualOS.Getwd()),
	// 	interp.Env(expand.ListEnviron(s.VirtualOS.Environ()...)),
	// 	interp.ExecHandler(func(ctx context.Context, args []string) error {
	// 		// exechandler
	// 		//hc := interp.HandlerCtx(ctx)
	//
	// 		// hc.Env
	// 		// hc.Dir
	// 		// hc.Stdin
	// 		// hc.Stdout
	// 		// hc.Stderr
	//
	// 		fmt.Printf("Running %q\n", args)
	// 		return interp.NewExitStatus(0)
	// 	}),
	// 	interp.OpenHandler(func(ctx context.Context, path string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	// 		return s.VirtualOS.OpenFile(path, flag, perm)
	// 	}),
	// 	// Params populates the shell options and parameters
	// 	// interp.Params()
	// 	interp.StdIO(s.VirtualOS.Stdin(), s.VirtualOS.Stdout(), s.VirtualOS.Stderr()),
	// )
	// if err != nil {
	// 	log.Println(err)
	// 	fmt.Fprintln(s.Readline, "-bash: syntax error: unexpected end of file")
	// 	return 1
	// }
	// runner.Reset()

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
			// Interrupt clears line.
			continue
		case err != nil:
			log.Printf("Error readline: %v", err)
			continue

		case len(line) == 0:
			continue // empty line

		default:
			prog, err := syntax.NewParser().Parse(strings.NewReader(line), "")
			if err != nil {
				fmt.Fprintf(s.Readline, "sh: syntax error: %v\n", err)
				continue
			}
			s.executeFile(prog)
		}
	}
	return 0
}

func (s *Shell) executeFile(file *syntax.File) {
	for _, stmt := range file.Stmts {
		s.executeStatement(execContext{}, stmt)
	}
}

type execContext struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	env []string
}

func (s *Shell) executeStatement(ec execContext, stmt *syntax.Stmt) error {
	// set up redirects

	// run command
	switch cmd := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		// if assign and no command -> set global env

		// evaluate assigns
		fmt.Fprintln(s.Readline, "call")
		syntax.DebugPrint(s.Readline, cmd)
		return nil
	case *syntax.BinaryCmd:
		switch cmd.Op {
		case syntax.AndStmt:
			if err := s.executeStatement(ec, cmd.X); err != nil {
				return err
			}
			if s.lastRet == 0 {
				return s.executeStatement(ec, cmd.Y)
			}
			return nil
		case syntax.OrStmt:
			if err := s.executeStatement(ec, cmd.X); err != nil {
				return err
			}
			if s.lastRet != 0 {
				return s.executeStatement(ec, cmd.Y)
			}
			return nil
		case syntax.Pipe:
			fmt.Fprintln(s.Readline, "PIPE")
			if err := s.executeStatement(ec, cmd.X); err != nil {
				return err
			}

			if err := s.executeStatement(ec, cmd.Y); err != nil {
				return err
			}
			return nil
		default:
			// Fail for unknown operations.
		}
	default:
		// Fail for other types of statements
	}

	return fmt.Errorf("syntax error near: %s", stmt.Pos())
}

func (s *Shell) evalAssign(ec execContext, assignments []*syntax.Assign) ([]string, error) {
	var out []string

	for _, assmt := range assignments {
		if assmt.Name == nil {
			continue
		}
		key := assmt.Name.Value
		var value string
		if word := assmt.Value; word != nil {
			for _, part := range word.Parts {
				switch part := part.(type) {
				case *syntax.Lit:
					value += part.Value
				case *syntax.ParamExp:
					param := part.Param
					if param == nil {
						fmt.Errorf("syntax error near: %s", param.Pos())
					}
					// lookup param.Value
				}
			}

			// word -> wordpart
		}

		value := assmt.Value
	}
	// A=B AA=$A$A echo $AA
	//
	// A=B AA=$A$A
	// echo $AA
	// BB
	//
	// A=B AA=$A$A env
	// A=B
	// AA=BB

	return out, nil
}

func (s *Shell) runInteractive() int {
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
			// Interrupt clears line.
			continue
		case err != nil:
			log.Printf("Error readline: %v", err)
			continue

		case len(line) == 0:
			continue // empty line

		default:
			s.runCommand(line)
		}
	}
	return 0
}

func (s *Shell) runCommand(command string) {
	tokens, err := shlex.Split(command, true)
	if err != nil {
		fmt.Fprintln(s.Readline, "sh: syntax error: unexpected end of file")
		return
	}
	if len(tokens) == 0 {
		return
	}

	// Take off command environment variables
	var assignments []string
	var cmdEnvStop int
	for ; cmdEnvStop < len(tokens); cmdEnvStop++ {
		tok := tokens[cmdEnvStop]
		if strings.Contains(tok, "=") {
			assignments = append(assignments, tok)
		} else {
			break
		}
	}

	tokens = tokens[cmdEnvStop:]

	// If the full command was environment variables, set them. Otherwise they
	// should only be populated for the upcoming command.
	if 0 == len(tokens) {
		vos.CopyEnv(s.VirtualOS, assignments)
		return
	}

	// Expand the environment
	for i, tok := range tokens {
		mapEnv := s.cmdEnv()
		vos.CopyEnv(mapEnv, assignments)
		tokens[i] = os.Expand(tok, mapEnv.Getenv)
	}

	s.ExecuteProgramOrBuiltin(append(s.VirtualOS.Environ(), assignments...), tokens)
}

// cmdEnv returns a new copy of the VOS environment with special variables set
// for shell expansion.
func (s *Shell) cmdEnv() vos.VEnv {
	mapEnv := vos.NewMapEnvFromEnvList(s.VirtualOS.Environ())

	// Shell only arguments
	mapEnv.Setenv("$", fmt.Sprintf("%d", s.VirtualOS.Getpid()))
	mapEnv.Setenv("?", fmt.Sprintf("%d", uint8(s.lastRet)))
	mapEnv.Setenv("WIDTH", fmt.Sprintf("%d", s.VirtualOS.GetPTY().Width))
	mapEnv.Setenv("HEIGHT", fmt.Sprintf("%d", s.VirtualOS.GetPTY().Height))

	return mapEnv
}

func (s *Shell) ExecuteProgramOrBuiltin(cmdEnv []string, args []string) {
	if len(args) == 0 {
		return
	}

	// Execute builtins
	if builtin, ok := AllBuiltins[args[0]]; ok {
		s.lastRet = builtin.Main(s, args)
		return
	}

	s.ExecuteProgram(cmdEnv, args)
}

func (s *Shell) ExecuteProgram(cmdEnv []string, args []string) {
	if len(args) == 0 {
		return
	}

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
