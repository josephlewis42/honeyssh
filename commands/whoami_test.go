package commands

import (
	"testing"
)

func TestWhoami(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"whoami"}},
		"help":   {[]string{"whoami", "--help"}},
	}

	cases.Run(t, Whoami)
}
