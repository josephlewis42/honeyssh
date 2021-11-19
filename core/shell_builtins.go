package core

import (
	"fmt"
	"strings"

	"github.com/pborman/getopt/v2"
)

// AllBuiltins holds a list of all registered shell builtins
var AllBuiltins = make(map[string]ShellBuiltin)

type ShellBuiltin interface {
	Main(s *Shell, args []string) int
}

type ShellBuiltinFunc func(s *Shell, args []string) int

func (f ShellBuiltinFunc) Main(s *Shell, args []string) int {
	return f(s, args)
}

var _ ShellBuiltin = (ShellBuiltinFunc)(nil)

func Unset(s *Shell, args []string) int {
	opts := getopt.New()
	opts.Bool('f', "treat NAME as a function")
	opts.Bool('v', "treat NAME as a variable")
	opts.Bool('n', "treat NAME as a reference")
	helpOpt := opts.BoolLong("help", 'h', "show help and exit")

	optErr := opts.Getopt(args, nil)
	w := s.VirtualOS.Stdout()

	if optErr != nil || *helpOpt {
		fmt.Fprintln(w, "usage: unset [-fvn] [NAME...]")
		fmt.Fprintln(w, "Unset shell values and functions.")
	}

	return 0
}

// Cd is the cd shell builtin
func Cd(s *Shell, args []string) int {
	switch len(args) {
	case 1:
		args = append(args, s.VirtualOS.Getenv(EnvHome))
		fallthrough
	case 2:
		if err := s.VirtualOS.Chdir(args[1]); err != nil {
			fmt.Fprintf(s.VirtualOS.Stderr(), "%s: %v\n", args[0], err)
			return 1
		}
	default:
		fmt.Fprintf(s.VirtualOS.Stderr(), "%s: too many arguments\n", args[0])
		return 1
	}
	return 0
}

// Exit quits the shell
func Exit(s *Shell, args []string) int {
	s.VirtualOS.SSHExit(0)
	return 0
}

func History(s *Shell, args []string) int {
	// parse -c to clear

	opts := getopt.New()
	clear := opts.Bool('c', "clear the history by deleting all entries")
	append := opts.Bool('a', "append all history to the hsitory file")
	helpOpt := opts.BoolLong("help", 'h', "show help and exit")

	if err := opts.Getopt(args, nil); err != nil || *helpOpt {
		w := s.VirtualOS.Stderr()
		if err != nil {
			fmt.Fprintln(w, err)
		}
		fmt.Fprintln(w, "Display or manipulate the history list")
		fmt.Fprintln(w, "Display the history list with line numbers.")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Options:")
		opts.PrintOptions(w)
		return 1
	}

	optionChosen := false
	if *clear {
		s.Readline.Operation.ResetHistory()
		s.history = nil
		optionChosen = true
	}
	if *append {
		// nop
		optionChosen = true
	}

	if !optionChosen {
		for i, line := range s.history {
			fmt.Fprintf(s.VirtualOS.Stdout(), "% 5d  %s\n", i, line)
		}
	}
	return 0
}

func Help(s *Shell, args []string) int {
	w := s.VirtualOS.Stdout()
	fmt.Fprintln(w, "sh version 4.31.20")
	fmt.Fprintln(w, "These shell commands are defined internally.  Type `help' to see this list.")
	fmt.Fprintln(w, "Type `help name' to find out more about the function `name'.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Builtins:")
	fmt.Fprintln(w)

	var builtins []string
	for k := range AllBuiltins {
		builtins = append(builtins, k)
	}

	fmt.Fprintln(w, strings.Join(builtins, "\n"))

	return 0
}

func init() {
	AllBuiltins["unset"] = ShellBuiltinFunc(Unset)
	AllBuiltins["cd"] = ShellBuiltinFunc(Cd)
	AllBuiltins["history"] = ShellBuiltinFunc(History)
	AllBuiltins["help"] = ShellBuiltinFunc(Help)
	AllBuiltins["exit"] = ShellBuiltinFunc(Exit)
}
