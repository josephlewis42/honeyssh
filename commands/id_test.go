package commands

import (
	"testing"
)

func TestId(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"id"}},
		"help":   {[]string{"id", "--help"}},
	}

	cases.Run(t, Id)
}
