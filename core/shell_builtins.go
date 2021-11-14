package core

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
