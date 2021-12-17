package commands

import "testing"

func TestTouch(t *testing.T) {
	AssertScript(t, "/bin/touch foo", "/bin/ls -lah")
}
