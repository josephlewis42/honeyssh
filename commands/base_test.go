package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sebdah/goldie/v2"
	"josephlewis.net/osshit/core/vos"
	"josephlewis.net/osshit/core/vos/vostest"
)

func ExampleBytesToHuman() {

	// < 1k is presented directly
	fmt.Println(BytesToHuman(512))

	// Multiples > 10 are shown without decimal.
	fmt.Println(BytesToHuman(23 * 10e8))

	// Multiples < 10 are shown with decimal.
	fmt.Println(BytesToHuman(5 * 1024))

	// Output: 512
	// 23G
	// 5.1K
}

func TestAllCommands(t *testing.T) {
	for _, cmdEntry := range ListBuiltinCommands() {
		t.Run(strings.Join(cmdEntry.Names, ","), func(t *testing.T) {
			if cmdEntry.Proc == nil {
				t.Fatal("nil command", cmdEntry.Names)
			}
		})
	}
}

type goldenTestSuite map[string]goldenTest

type goldenTest struct {
	Args []string
}

func (gts goldenTestSuite) Run(t *testing.T, cmd vos.ProcessFunc) {
	t.Helper()

	g := goldie.New(
		t,
		goldie.WithFixtureDir(filepath.Join("testdata", "golden")),
		goldie.WithDiffEngine(goldie.ColoredDiff),
		goldie.WithTestNameForDir(true),
	)

	for tn, tc := range gts {
		t.Run(tn, func(t *testing.T) {
			cmd := vostest.Command(cmd, tc.Args[0], tc.Args[1:]...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}

			g.Assert(t, tn, out)
		})
	}
}
