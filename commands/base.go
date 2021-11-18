package commands

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

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

func BytesToHuman(bytes int64) string {
	for _, e := range []struct {
		unit  string
		power int64
	}{
		{"P", 1e15},
		{"T", 1e12},
		{"G", 1e9},
		{"M", 1e6},
		{"K", 1e3},
	} {
		quotient := bytes / e.power
		switch {
		case quotient == 0:
			continue
		case quotient > 10:
			return fmt.Sprintf("%d%s", quotient, e.unit)
		default:
			return fmt.Sprintf("%0.1f%s", float64(bytes)/float64(e.power), e.unit)
		}
	}

	return fmt.Sprintf("%d", bytes)
}

func UidResolver(virtOS vos.VOS) (resolver func(int) string) {
	mapping := map[int]string{
		0: "root", // seed in case we don't see any others.
	}

	resolver = func(uid int) string {
		if resolved, ok := mapping[uid]; ok {
			return resolved
		}
		return fmt.Sprintf("%d", uid)
	}

	fd, err := virtOS.Open("/etc/passwd")
	if err != nil {
		virtOS.LogInvalidInvocation(err)
		return
	}

	passwdBytes, err := ioutil.ReadAll(fd)
	if err != nil {
		// can't do anything
		return
	}
	passwdFile := string(passwdBytes)
	for _, line := range strings.Split(passwdFile, "\n") {
		entry := strings.Split(line, ":")
		if len(entry) < 3 {
			continue
		}
		// name:X:uid:
		name := entry[0]
		if uid, err := strconv.Atoi(entry[2]); err == nil {
			mapping[uid] = name
		}
	}

	return
}
