package commands

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/abiosoft/readline"
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

		return s.runInteractive()
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

func (s *Shell) prompt() string {
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

func (s *Shell) logSyntaxError(node syntax.Node) error {
	buf := &bytes.Buffer{}
	syntax.DebugPrint(buf, node)
	s.VirtualOS.LogInvalidInvocation(fmt.Errorf("sh syntax error: %s", buf.String()))

	return fmt.Errorf("syntax error near: %d", node.Pos().Col())
}

func (s *Shell) executeFile(file *syntax.File) error {
	for _, stmt := range file.Stmts {
		ec := execContext{
			stdin:  s.VirtualOS.Stdin(),
			stdout: s.VirtualOS.Stdout(),
			stderr: s.VirtualOS.Stderr(),
			env:    s.cmdEnv().Environ(),
		}
		if err := s.executeStatement(ec, stmt); err != nil {
			return err
		}
	}
	return nil
}

type execContext struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// env contains the shell environment variables in the execution context,
	// these contain pseudo-environment variables that aren't suitable to write
	// back to the system env like $$ and $@
	env []string

	// assignments contains command environment vrariable assignments.
	assignments []string

	// args contains the CLI arguments for the command
	args []string
}

func (s *Shell) executeStatement(ec execContext, stmt *syntax.Stmt) error {
	for _, redirect := range stmt.Redirs {
		// Only support output indirection (>)
		if redirect.Op != syntax.RdrOut && redirect.Op != syntax.DplOut {
			return s.logSyntaxError(redirect)
		}

		from := ""
		if redirect.N != nil {
			from = redirect.N.Value
		}

		var fromWriter *io.Writer
		switch from {
		case "", "1": // stdout
			fromWriter = &ec.stdout
		case "2": // stderr
			fromWriter = &ec.stderr
		default:
			return s.logSyntaxError(redirect)
		}

		if redirect.Word == nil {
			return s.logSyntaxError(redirect)
		}
		to, err := s.evalWord(ec, redirect.Word)
		if err != nil {
			return err
		}
		switch {
		case to == "":
			return s.logSyntaxError(redirect)
		case redirect.Op == syntax.DplOut && to == "1":
			*fromWriter = ec.stdout
		case redirect.Op == syntax.DplOut && to == "2":
			*fromWriter = ec.stderr
		default:
			fd, err := s.VirtualOS.Create(to)
			if err != nil {
				return err
			}
			defer fd.Close()
			*fromWriter = fd
		}
	}

	// run command
	switch cmd := stmt.Cmd.(type) {
	case *syntax.CallExpr:
		var err error
		ec.assignments, err = s.evalAssign(ec, cmd.Assigns)
		if err != nil {
			return err
		}

		for _, word := range cmd.Args {
			argStr, err := s.evalWord(ec, word)
			if err != nil {
				return err
			}
			ec.args = append(ec.args, argStr)
		}
		s.executeProgramOrBuiltin(ec)
	case *syntax.BinaryCmd:
		switch cmd.Op {
		case syntax.AndStmt:
			if err := s.executeStatement(ec, cmd.X); err != nil {
				return err
			}
			if s.lastRet == 0 {
				return s.executeStatement(ec, cmd.Y)
			}
		case syntax.OrStmt:
			if err := s.executeStatement(ec, cmd.X); err != nil {
				return err
			}
			if s.lastRet != 0 {
				return s.executeStatement(ec, cmd.Y)
			}
		case syntax.Pipe:
			buf := &bytes.Buffer{}
			xEc := ec
			xEc.stdout = buf
			if err := s.executeStatement(xEc, cmd.X); err != nil {
				return err
			}

			yEc := ec
			yEc.stdin = buf
			if err := s.executeStatement(yEc, cmd.Y); err != nil {
				return err
			}
		default:
			// Fail for unknown operations.
			return s.logSyntaxError(stmt)
		}
	default:
		// Fail for other types of statements
		return s.logSyntaxError(stmt)
	}

	return nil
}

func (s *Shell) evalAssign(ec execContext, assignments []*syntax.Assign) ([]string, error) {
	out := vos.NewMapEnv()
	tmpEnv := vos.NewMapEnvFromEnvList(ec.env)

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
						return nil, s.logSyntaxError(word)
					}
					value += tmpEnv.Getenv(param.Value)
				default:
					return nil, s.logSyntaxError(word)
				}
			}
		}

		tmpEnv.Setenv(key, value)
		out.Setenv(key, value)
	}

	return out.Environ(), nil
}

func (s *Shell) evalWord(ec execContext, word *syntax.Word) (string, error) {
	if word == nil {
		return "", nil
	}
	var out []string

	for _, part := range word.Parts {
		subEval, err := s.evalWordPart(ec, part)
		if err != nil {
			return "", err
		}
		out = append(out, subEval)
	}
	return strings.Join(out, ""), nil
}

func (s *Shell) evalWordPart(ec execContext, part syntax.WordPart) (string, error) {
	switch part := part.(type) {
	case *syntax.Lit:
		return part.Value, nil

	case *syntax.SglQuoted:
		return part.Value, nil

	case *syntax.DblQuoted:
		var out []string
		for _, subPart := range part.Parts {
			subEval, err := s.evalWordPart(ec, subPart)
			if err != nil {
				return "", err
			}
			out = append(out, subEval)
		}
		return strings.Join(out, ""), nil

	case *syntax.ParamExp:
		param := part.Param
		if param == nil {
			return "", s.logSyntaxError(part)
		}
		tmpEnv := vos.NewMapEnvFromEnvList(ec.env)
		return tmpEnv.Getenv(param.Value), nil

	default:
		return "", s.logSyntaxError(part)
	}
}

func (s *Shell) runInteractive() int {
	for !s.Quit {
		s.Readline.SetPrompt(s.prompt())
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

func (s *Shell) runCommand(line string) {
	prog, err := syntax.NewParser().Parse(strings.NewReader(line), "")
	if err != nil {
		fmt.Fprintf(s.Readline, "sh: syntax error: %v\n", err)
		return
	}
	if err := s.executeFile(prog); err != nil {
		fmt.Fprintf(s.Readline, "sh: %v\n", err)
	}
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

func (s *Shell) executeProgramOrBuiltin(ec execContext) {
	if len(ec.args) == 0 {
		// If the full command was environment variables, set them. Otherwise they
		// should only be populated for the upcoming command.
		vos.CopyEnv(s.VirtualOS, ec.assignments)
		return
	}

	// Execute builtins
	if builtin, ok := AllBuiltins[ec.args[0]]; ok {
		s.lastRet = builtin.Main(s, ec.args)
		return
	}

	// Execute program
	proc, err := s.VirtualOS.StartProcess(ec.args[0], ec.args, &vos.ProcAttr{
		Env:   append(s.VirtualOS.Environ(), ec.assignments...),
		Files: vos.NewVIOAdapter(ec.stdin, ec.stdout, ec.stderr),
	})
	if err != nil {
		fmt.Fprintf(s.Readline, "sh: %s\n", err)
		return
	}

	s.lastRet = proc.Run()
}

func init() {
	mustAddBinCmd("sh", RunShell)
}
