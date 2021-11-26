package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"

	"josephlewis.net/osshit/core/vos"
)

const (
	ModeMaskUser  fs.FileMode = 0700
	ModeMaskGroup             = 0070
	ModeMaskOther             = 0007
	ModeMaskAll               = ModeMaskUser | ModeMaskGroup | ModeMaskOther

	ModeRead  fs.FileMode = 0444
	ModeWrite             = 0222
	ModeExec              = 0111

	ChmodMask = ModeMaskAll
)

func blendChmod(origValue, newValue fs.FileMode) fs.FileMode {
	return (origValue &^ ChmodMask) | (newValue & ChmodMask)
}

func ChmodApplyMode(mode string, orig fs.FileMode) (fs.FileMode, error) {

	// If mode is an octal integer, the value is absolute
	if octalMode, err := strconv.ParseUint(mode, 8, 32); err == nil {
		return blendChmod(orig, fs.FileMode(octalMode)), nil
	}

	var who fs.FileMode
	var apply fs.FileMode
	var action func(orig, who, apply fs.FileMode) fs.FileMode

	// This is a simplified algorithm that doesn't handle the full grammar or
	// semantics but should be good enough to pass a sniff test.
	for _, modeChar := range mode {
		switch modeChar {
		// Mask groups
		case 'a':
			who |= ModeMaskAll
		case 'u':
			who |= ModeMaskUser
		case 'g':
			who |= ModeMaskGroup
		case 'o':
			who |= ModeMaskOther
		case '+':
			action = func(orig, who, apply fs.FileMode) fs.FileMode {
				return blendChmod(orig, orig|(apply&who))
			}
		case '=':
			action = func(orig, who, apply fs.FileMode) fs.FileMode {
				return blendChmod(orig, (apply & who))
			}
		case '-':
			action = func(orig, who, apply fs.FileMode) fs.FileMode {
				return blendChmod(orig, orig & ^(apply&who))
			}
		case 'r':
			apply |= ModeRead
		case 'w':
			apply |= ModeWrite
		case 'x':
			apply |= ModeExec
		case 'X':
			if (who&ModeExec) > 0 || (orig&fs.ModeDir) > 0 {
				apply |= ModeExec
			}
		case 's', 't':
			// Not implemneted
		default:
			return orig, fmt.Errorf("unknown symbol %q", modeChar)
		}
	}

	if action == nil {
		return orig, errors.New("no action provided")
	}

	if who == 0 {
		who = ModeMaskAll
	}

	return action(orig, who, apply), nil
}

// Chmod implements a POSIX chmod command.
func Chmod(virtOS vos.VOS) int {
	cmd := &SimpleCommand{
		Use:   "chmod [OPTION...] MODE FILE...",
		Short: "Change the mode of each FILE to MODE.",
	}

	args := virtOS.Args()
	if len(args) < 3 {
		fmt.Fprintln(virtOS.Stderr(), "chmod: not enough arguments")
		cmd.PrintHelp(virtOS.Stdout())
		return 1
	}

	modeExpr := args[1]
	paths := args[2:]

	var anyFailed bool
	for _, path := range paths {
		stat, err := virtOS.Stat(path)
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "chmod: couldn't stat %s: %v\n", path, err)
			anyFailed = true
			continue
		}

		newMode, err := ChmodApplyMode(modeExpr, stat.Mode())
		if err != nil {
			fmt.Fprintf(virtOS.Stderr(), "chmod: %s\n", err.Error())
			return 1
		}

		if err := virtOS.Chmod(path, newMode); err != nil {
			fmt.Fprintf(virtOS.Stderr(), "chmod: couldn't update %s\n", path)
			anyFailed = true
		}
	}

	if anyFailed {
		return 1
	}
	return 0
}

var _ HoneypotCommandFunc = Chmod

func init() {
	addBinCmd("chmod", HoneypotCommandFunc(Chmod))
}
