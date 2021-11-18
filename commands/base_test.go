package commands

import "fmt"

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
