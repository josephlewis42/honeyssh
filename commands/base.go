package commands

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
	"strconv"
	"strings"

	"github.com/fatih/color"
	fcolor "github.com/fatih/color"
	"github.com/josephlewis42/honeyssh/core/vos"
	getopt "github.com/pborman/getopt/v2"
)

// Always generate golden files on generate, a diff indicates a problem and
// should be caught at code review or commit time.
//go:generate go test . -update

type CommandEntry struct {
	Names []string
	Proc  vos.ProcessFunc
}

type commandTable struct {
	commands []CommandEntry
	lookup   map[string]vos.ProcessFunc
}

func (ct *commandTable) AddCommand(proc vos.ProcessFunc, names ...string) {
	ct.commands = append(ct.commands, CommandEntry{
		Names: names,
		Proc:  proc,
	})
	if ct.lookup == nil {
		ct.lookup = make(map[string]vos.ProcessFunc)
	}
	for _, name := range names {
		ct.lookup[name] = proc
	}
}

var allCommands = commandTable{}

// BuiltinProcessResolver implemnts vos.ProcessResolver, it returns the builtin
// command with the given path or nil if none exists.
func BuiltinProcessResolver(command string) vos.ProcessFunc {
	return allCommands.lookup[command]
}

func ListBuiltinCommands() []CommandEntry {
	return allCommands.commands
}

var _ vos.ProcessResolver = BuiltinProcessResolver

// addBinCmd adds a command under /bin and /usr/bin.
func addBinCmd(name string, cmd vos.ProcessFunc) {
	allCommands.AddCommand(cmd, path.Join("/bin", name), path.Join("/usr/bin", name))
}

// addSbinCmd adds a command under /sbin and /usr/sbin.
func addSbinCmd(name string, cmd vos.ProcessFunc) {
	allCommands.AddCommand(cmd, path.Join("/sbin", name), path.Join("/usr/sbin", name))
}

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

	if err != nil && !s.NeverBail {
		fmt.Fprintf(virtOS.Stderr(), "error: %s\n\n", err)

		s.PrintHelp(virtOS.Stdout())
		return 1
	}

	if *s.ShowHelp {
		s.PrintHelp(virtOS.Stdout())
		return 0
	}

	return callback()
}

// RunEachArg runs the callback for every supplied arg.
func (s *SimpleCommand) RunEachArg(virtOS vos.VOS, callback func(string) error) int {
	return s.Run(virtOS, func() int {
		anyErrored := false

		for _, arg := range s.Flags().Args() {
			if err := callback(arg); err != nil {
				s.LogProgramError(virtOS, err)
				anyErrored = true
			}
		}

		if anyErrored {
			return 1
		}
		return 0
	})
}

// RunE runs the callback and converts the error into a return value and message.
func (s *SimpleCommand) RunE(virtOS vos.VOS, callback func() error) int {
	return s.Run(virtOS, func() int {
		if err := callback(); err != nil {
			s.LogProgramError(virtOS, err)
			return 1
		}
		return 0
	})
}

// RunEachFileOrStdin runs the callback for every supplied arg, or stdin
func (s *SimpleCommand) RunEachFileOrStdin(virtOS vos.VOS, files []string, callback func(name string, fd io.Reader) error) int {
	return s.Run(virtOS, func() int {
		anyErrored := false

		openCallback := func(name string) error {
			fd, err := virtOS.Open(name)
			if err != nil {
				return err
			}

			defer fd.Close()
			return callback(name, fd)
		}

		for _, arg := range files {
			if err := openCallback(arg); err != nil {
				s.LogProgramError(virtOS, err)
				anyErrored = true
			}

		}

		if len(files) == 0 {
			if err := callback("-", virtOS.Stdin()); err != nil {
				s.LogProgramError(virtOS, err)
				anyErrored = true
			}
		}

		if anyErrored {
			return 1
		}
		return 0
	})
}

// Log a program error to stderr in the form "program name: error message"
func (s *SimpleCommand) LogProgramError(virtOS vos.VOS, err error) {
	fmt.Fprintf(virtOS.Stderr(), "%s: %s\n", s.Flags().Program(), err.Error())
}

const (
	colorAlways = "always"
	colorAuto   = "auto"
	colorNever  = "never"
)

var (
	ColorBoldBlue  = color.New(color.FgBlue, color.Bold)
	ColorBoldGreen = color.New(color.FgGreen, color.Bold)
	ColorBoldCyan  = fcolor.New(color.FgCyan, color.Bold)
	ColorBoldRed   = color.New(color.FgRed, color.Bold)
)

type ColorPrinter struct {
	value  *string
	virtOS vos.VOS
}

// Init sets up the flag and virtual OS to determine the color output.
func (c *ColorPrinter) Init(flags *getopt.Set, virtOS vos.VOS) {
	c.virtOS = virtOS
	c.value = flags.EnumLong(
		"color",
		rune(0), // No short flag.
		[]string{colorAlways, colorAuto, colorNever},
		colorAuto,
		"colorize the output (always|auto|never)")
}

func (c *ColorPrinter) ShouldColor() bool {
	switch {
	case *c.value == colorNever:
		return false
	case *c.value == colorAlways:
		return true
	default:
		return c.virtOS.GetPTY().IsPTY
	}
}

func (c *ColorPrinter) applyIfShouldColor(formatter func(string, ...interface{}) string, format string, a ...interface{}) string {
	if c.ShouldColor() {
		return formatter(format, a...)
	}
	return fmt.Sprintf(format, a...)
}

func (c *ColorPrinter) Sprintf(color *color.Color, format string, a ...interface{}) string {
	if c.ShouldColor() {
		return color.Sprintf(format, a...)
	}
	return fmt.Sprintf(format, a...)
}
