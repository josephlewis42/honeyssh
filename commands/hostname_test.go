package commands

import (
	"testing"
)

func TestHostname(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg": {[]string{"hostname"}},
		"help":   {[]string{"hostname", "--help"}},
	}

	cases.Run(t, Hostname)
}
