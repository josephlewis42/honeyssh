package commands

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/josephlewis42/honeyssh/core/vos/vostest"
)

func TestCat(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg":  {[]string{"cat"}},
		"help":    {[]string{"cat", "--help"}},
		"missing": {[]string{"cat", "does not exist.txt"}},
	}

	cases.Run(t, Cat)
}

func TestCat_files(t *testing.T) {
	cmd := vostest.Command(Cat, "cat", "/foo.txt")

	// Test with missing file
	{
		assert.Nil(t, cmd.Run())

		assert.NotEqual(t, 0, cmd.ExitStatus, "exit code")
	}
	{
		// Create file and
		helloWorld := []byte("Hello, world!")
		assert.Nil(t, afero.WriteFile(cmd.VOS, "/foo.txt", helloWorld, 0600))

		out, err := cmd.CombinedOutput()

		assert.Equal(t, 0, cmd.ExitStatus, "exit code")
		assert.Nil(t, err)
		assert.Equal(t, string(helloWorld), string(out))
	}
}
