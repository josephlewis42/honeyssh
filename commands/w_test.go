package commands

import (
	"testing"
)

func TestW(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"w"}},
		"help":   {[]string{"w", "--help"}},
	}

	cases.Run(t, W)
}
