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
	getopt "github.com/pborman/getopt/v2"
	"josephlewis.net/osshit/core/vos"
)

type HoneypotCommandFunc = vos.ProcessFunc

// AllCommands holds a list of all registered commands
var AllCommands = make(map[string]HoneypotCommandFunc)

// addBinCmd adds a command under /bin and /usr/bin.
func addBinCmd(name string, cmd HoneypotCommandFunc) {
	AllCommands[path.Join("/bin", name)] = cmd
	AllCommands[path.Join("/usr/bin", name)] = cmd
}

// addSbinCmd adds a command under /sbin and /usr/sbin.
func addSbinCmd(name string, cmd vos.ProcessFunc) {
	AllCommands[path.Join("/sbin", name)] = cmd
	AllCommands[path.Join("/usr/sbin", name)] = cmd
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
