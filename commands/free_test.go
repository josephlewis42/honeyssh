package commands

import (
	"testing"
)

func TestFree(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"free"}},
		"help":   {[]string{"free", "--help"}},
		"human":  {[]string{"free", "-h"}},
	}

	cases.Run(t, Free)
}
