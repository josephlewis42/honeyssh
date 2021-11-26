package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"
)

func TestChmodApplyMode(t *testing.T) {
	blank := fs.FileMode(0)
	file := fs.FileMode(0666)

	cases := []struct {
		orig     fs.FileMode
		mode     string
		wantMode fs.FileMode
		wantErr  error
	}{
		// Permissions
		{blank, "+r", ModeRead, nil},
		{blank, "+w", ModeWrite, nil},
		{blank, "+x", ModeExec, nil},
		{blank, "+rwx", fs.FileMode(0777), nil},

		// No-op permissions
		{blank, "+t", blank, nil},
		{blank, "+s", blank, nil},

		// Capital X, only sets execute if a dir or already has an exec bit
		{blank, "+X", blank, nil},
		{fs.ModeDir, "+X", fs.ModeDir | ModeExec, nil},

		// Groups: a,u,g,o
		{blank, "a+r", ModeRead, nil},
		{blank, "a+w", ModeWrite, nil},
		{blank, "a+x", ModeExec, nil},
		{blank, "a+rwx", fs.FileMode(0777), nil},
		{blank, "u+r", ModeRead & ModeMaskUser, nil},
		{blank, "u+w", ModeWrite & ModeMaskUser, nil},
		{blank, "u+x", ModeExec & ModeMaskUser, nil},
		{blank, "u+rwx", fs.FileMode(0777) & ModeMaskUser, nil},
		{blank, "g+r", ModeRead & ModeMaskGroup, nil},
		{blank, "g+w", ModeWrite & ModeMaskGroup, nil},
		{blank, "g+x", ModeExec & ModeMaskGroup, nil},
		{blank, "g+rwx", fs.FileMode(0777) & ModeMaskGroup, nil},
		{blank, "o+r", ModeRead & ModeMaskOther, nil},
		{blank, "o+w", ModeWrite & ModeMaskOther, nil},
		{blank, "o+x", ModeExec & ModeMaskOther, nil},
		{blank, "o+rwx", fs.FileMode(0777) & ModeMaskOther, nil},

		// Actions:
		{ModeWrite | ModeRead, "-w", ModeRead, nil},
		{fs.FileMode(0777), "=r", ModeRead, nil},

		// Octal permissions
		{blank, "644", fs.FileMode(0644), nil},

		// Don't wipe non-permission bits
		{fs.ModeDir | fs.ModeSticky, "+x", fs.ModeDir | fs.ModeSticky | ModeExec, nil},
		{fs.ModeDir | fs.ModeSticky, "-x", fs.ModeDir | fs.ModeSticky, nil},
		{fs.ModeDir | fs.ModeSticky, "=x", fs.ModeDir | fs.ModeSticky | ModeExec, nil},
		{fs.ModeDir | fs.ModeSticky, "644", fs.ModeDir | fs.ModeSticky | fs.FileMode(0644), nil},

		// Bad mode expressions
		{file, "o+z", file, errors.New("unknown symbol 'z'")},
		{file, "x", file, errors.New("no action provided")},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("chmod %q %q to %q %v", tc.mode, tc.orig, tc.wantMode, tc.wantErr), func(t *testing.T) {

			gotMode, gotErr := ChmodApplyMode(tc.mode, tc.orig)
			if tc.wantErr != nil || gotErr != nil {
				if tc.wantErr.Error() != gotErr.Error() {
					t.Errorf("wanted err %q got err %q", tc.wantErr, gotErr)
				}
			}

			if gotMode != tc.wantMode {
				t.Errorf("wanted mode %q got mode %q", tc.wantMode, gotMode)
			}
		})
	}
}
