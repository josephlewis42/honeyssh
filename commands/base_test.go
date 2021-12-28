package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anmitsu/go-shlex"
	"github.com/sebdah/goldie/v2"
	"github.com/josephlewis42/honeyssh/core/vos"
	"github.com/josephlewis42/honeyssh/core/vos/vostest"
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

type stepOutput struct {
	Command []string `json:"command"`
	Output  string   `json:"output"`
}

func AssertScript(t *testing.T, commands ...string) {
	t.Helper()

	fakeProc := func(vos vos.VOS) int {
		p := BuiltinProcessResolver(vos.Args()[0])
		if p == nil {
			t.Fatalf("couldn't resolve process: %q", vos.Args()[0])
		}
		return p(vos)
	}

	var steps []stepOutput

	vosCommand := vostest.Command(fakeProc, "")
	for _, command := range commands {
		split, err := shlex.Split(command, true)
		if err != nil {
			t.Fatalf("couldn't parse %q: %v", command, err)
		}
		vosCommand.Argv = split

		out, err := vosCommand.CombinedOutput()
		if err != nil {
			t.Fatalf("couldn't run %q: %v", command, err)
		}

		if vosCommand.ExitStatus != 0 {
			t.Fatalf("%q failed with exit status: %d", command, vosCommand.ExitStatus)
		}

		steps = append(steps, stepOutput{
			Command: split,
			Output:  string(out),
		})
	}

	g := goldie.New(
		t,
		goldie.WithFixtureDir(filepath.Join("testdata", "golden")),
		goldie.WithDiffEngine(goldie.ColoredDiff),
		goldie.WithTestNameForDir(true),
	)

	g.AssertJson(t, t.Name(), steps)
}
