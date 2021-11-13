package commands

import (
	"path"

	"josephlewis.net/osshit/core/vos"
)

// AllCommands holds a list of all registered commands
var AllCommands = make(map[string]HoneypotCommand)

func addBinCmd(name string, cmd HoneypotCommand) {
	AllCommands[path.Join("/bin", name)] = cmd
	AllCommands[path.Join("/usr/bin", name)] = cmd
}

func addSbinCmd(name string, cmd HoneypotCommand) {
	AllCommands[path.Join("/sbin", name)] = cmd
	AllCommands[path.Join("/usr/sbin", name)] = cmd
}

type HoneypotCommand interface {
	Main(os vos.VOS) int
}

type HoneypotCommandFunc func(os vos.VOS) int

func (f HoneypotCommandFunc) Main(os vos.VOS) int {
	return f(os)
}

var _ HoneypotCommand = (HoneypotCommandFunc)(nil)
