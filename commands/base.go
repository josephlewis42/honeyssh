package commands

import (
	"path"

	"josephlewis.net/osshit/core/vos"
)

// AllCommands holds a list of all registered commands
var AllCommands = make(map[string]HoneypotCommand)

// addBinCmd adds a command under /bin and /usr/bin.
func addBinCmd(name string, cmd HoneypotCommand) {
	AllCommands[path.Join("/bin", name)] = cmd
	AllCommands[path.Join("/usr/bin", name)] = cmd
}

// addSbinCmd adds a command under /sbin and /usr/sbin.
func addSbinCmd(name string, cmd HoneypotCommand) {
	AllCommands[path.Join("/sbin", name)] = cmd
	AllCommands[path.Join("/usr/sbin", name)] = cmd
}

// A command that can be run by the Honeypot.
type HoneypotCommand interface {
	Main(virtualOS vos.VOS) int
}

// A function adapter for Honeypot commands.
type HoneypotCommandFunc func(os vos.VOS) int

func (f HoneypotCommandFunc) Main(os vos.VOS) int {
	return f(os)
}

var _ HoneypotCommand = (HoneypotCommandFunc)(nil)
