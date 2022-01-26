package commands

import (
	"testing"
)

func TestRunShell(t *testing.T) {
	cases := goldenTestSuite{
		"help":     {[]string{"sh", "--help"}},
		"echo":     {[]string{"sh", "-c", `/bin/echo "hello"`}},
		"echo-cat": {[]string{"sh", "-c", `/bin/echo "hello" > foo; /bin/cat foo`}},

		// Ensure environment expansion works as expected:
		"no-expand-args":     {[]string{"sh", "-c", `A=B AA=$A$A /bin/echo $AA`}},  // ""
		"expand-after-set":   {[]string{"sh", "-c", `A=B AA=$A$A; /bin/echo $AA`}}, // "BB"
		"expand-within-prog": {[]string{"sh", "-c", `A=B AA=$A$A /bin/env`}},

		// Redirects
		"redir-stdout-stderr": {[]string{"sh", "-c", `/bin/echo "hello" 1>&2`}},
		"redir-stderr-stdout": {[]string{"sh", "-c", `/bin/echo "hello" 2>&1`}},
		"redir-dev-null":      {[]string{"sh", "-c", `/bin/echo "hello" > /null`}},
		"redir-out-err-file":  {[]string{"sh", "-c", `/bin/echo "hello" 1>&2 2>tmp; /bin/cat tmp`}},
		"redir-invalid-file":  {[]string{"sh", "-c", `/bin/echo "hello" >/does/not/exist`}},

		// Pipes
		"pipe-shell": {[]string{"sh", "-c", `/bin/echo "/bin/w" | /bin/sh`}},

		// Syntax errors
		"err-redir-all":  {[]string{"sh", "-c", `/bin/env >>1`}},
		"err-bad-from":   {[]string{"sh", "-c", `/bin/env 3>&1`}},
		"err-blank-dest": {[]string{"sh", "-c", `/bin/env >''`}},
	}

	cases.Run(t, RunShell)
}
