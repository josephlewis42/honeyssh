package vos

import "fmt"

func ExampleCopyEnv() {
	env := NewMapEnv()
	CopyEnv(env, []string{"A=B", "C=D", "E", "F=G=H"})

	fmt.Printf("Environ(): %q\n", env.Environ())
	fmt.Printf("Getenv(\"F\"): %q\n", env.Getenv("F"))

	// Output: Environ(): ["A=B" "C=D" "E=" "F=G=H"]
	// Getenv("F"): "G=H"
}

func ExampleNewMapEnvFromEnvList() {
	env := NewMapEnvFromEnvList([]string{"A=B", "C=D", "E", "F=G=H"})

	fmt.Printf("Environ(): %q\n", env.Environ())
	fmt.Printf("Getenv(\"F\"): %q\n", env.Getenv("F"))

	// Output: Environ(): ["A=B" "C=D" "E=" "F=G=H"]
	// Getenv("F"): "G=H"
}

func ExampleMapEnv_Unsetenv() {
	env := NewMapEnv()
	env.Setenv("A", "B")
	env.Setenv("C", "D")

	fmt.Println("Before:", env.Environ())
	env.Unsetenv("A")
	fmt.Println("After:", env.Environ())

	// Output: Before: [A=B C=D]
	// After: [C=D]
}

func ExampleMapEnv_LookupEnv() {
	env := NewMapEnv()
	env.Setenv("A", "B")

	val, ok := env.LookupEnv("A")
	fmt.Println("Existing", "val:", val, "ok:", ok)
	val, ok = env.LookupEnv("B")
	fmt.Println("Missing", "val:", val, "ok:", ok)

	// Output: Existing val: B ok: true
	// Missing val:  ok: false
}
