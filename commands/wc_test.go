package commands

import (
	"testing"

	"github.com/josephlewis42/honeyssh/core/vos/vostest"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestWc(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg":  {[]string{"wc"}},
		"help":    {[]string{"wc", "--help"}},
		"missing": {[]string{"wc", "does not exist.txt"}},
	}

	cases.Run(t, Wc)
}

func TestWc_single_file(t *testing.T) {
	cmd := vostest.Command(Wc, "wc", "/foo.txt")

	// Test with missing file
	{
		assert.Nil(t, cmd.Run())

		assert.NotEqual(t, 0, cmd.ExitStatus, "exit code")
	}
	{
		// Create file and
		helloWorld := []byte("Hello,\nworld !")
		assert.Nil(t, afero.WriteFile(cmd.VOS, "/foo.txt", helloWorld, 0600))

		out, err := cmd.CombinedOutput()

		assert.Equal(t, 0, cmd.ExitStatus, "exit code")
		assert.Nil(t, err)
		assert.Equal(t, "1 3 14 /foo.txt\n", string(out))
	}
}
