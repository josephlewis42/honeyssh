package commands

import (
	"testing"
)

func TestUptime(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"uptime"}},
		"help":   {[]string{"uptime", "--help"}},
	}

	cases.Run(t, Uptime)
}
