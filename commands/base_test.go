package commands

import (
	"fmt"
	"testing"
)

func ExampleBytesToHuman() {

	// < 1k is presented directly
	fmt.Println(BytesToHuman(512))

	// Multiples > 10 are shown without decimal.
	fmt.Println(BytesToHuman(23 * 10e8))

	// Multiples < 10 are shown with decimal.
	fmt.Println(BytesToHuman(5 * 1024))

	// Output: 512
	// 23G
	// 5.1K
}

func TestAllCommands(t *testing.T) {
	for cn, cmd := range AllCommands {
		t.Run(cn, func(t *testing.T) {
			if cmd == nil {
				t.Fatal("nil command", cn)
			}

			// cmdName := path.Base(cn)

		})
	}
}
