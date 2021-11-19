package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	getopt "github.com/pborman/getopt/v2"
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

type SimpleCommand struct {
	// Use holds a one line usage string
	Use string
	// Short holds a sone line description of the command.
	Short string
	// ShowHelp sets whether help is displayed or not.
	// If this is non-nil when Run() is called, then the default help flag isn't
	// added.
	ShowHelp *bool
	// NeverBail skips interacting with stdout/stderr on failure and
	// always runs the callback.
	NeverBail bool

	flags *getopt.Set
}

// Flags gets the command's flag set.
func (s *SimpleCommand) Flags() *getopt.Set {
	if s.flags == nil {
		s.flags = getopt.New()
	}

	return s.flags
}

// PrintHelp writes help for the command to the given writer.
func (s *SimpleCommand) PrintHelp(w io.Writer) {
	fmt.Fprint(w, "usage: ")
	fmt.Fprintln(w, s.Use)
	fmt.Fprintln(w, s.Short)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	s.Flags().PrintOptions(w)
}

// Run the command, if flag parsing was succcessful call the callback.
func (s *SimpleCommand) Run(virtOS vos.VOS, callback func() int) int {
	opts := s.Flags()

	// Add help flag if not overridden.
	if s.ShowHelp == nil {
		s.ShowHelp = opts.BoolLong("help", 'h', "show this help and exit")
	}

	err := opts.Getopt(virtOS.Args(), nil)
	if err != nil {
		virtOS.LogInvalidInvocation(err)
	}

	if !s.NeverBail {
		if err != nil || *s.ShowHelp {
			if err != nil {
				fmt.Fprintf(virtOS.Stderr(), "error: %s\n\n", err)
			}

			s.PrintHelp(virtOS.Stdout())
			return 1
		}
	}

	return callback()
}
