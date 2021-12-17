package commands

import (
	"testing"
)

func TestUname(t *testing.T) {
	cases := goldenTestSuite{
		"no-arg":  {[]string{"uname"}},
		"help":    {[]string{"uname", "--help"}},
		"all":     {[]string{"uname", "-a"}},
		"kernel":  {[]string{"uname", "-srv"}},
		"node":    {[]string{"uname", "-n"}},
		"machine": {[]string{"uname", "-m"}},
		"invalid": {[]string{"uname", "-z"}},
	}

	cases.Run(t, Uname)
}
