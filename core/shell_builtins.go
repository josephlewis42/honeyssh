package core

import (
	"fmt"

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

// Nop is a shell builtin that does nothing.
func Nop(s *Shell, args []string) int {
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

func init() {
	for _, name := range []string{
		"unset",
	} {
		AllBuiltins[name] = ShellBuiltinFunc(Nop)
	}

	AllBuiltins["cd"] = ShellBuiltinFunc(Cd)
	AllBuiltins["history"] = ShellBuiltinFunc(History)
}
