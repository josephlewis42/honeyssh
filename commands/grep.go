package commands

import (
	"context"
	"fmt"

	"github.com/josephlewis42/honeyssh/core/vos"
	grep "github.com/josephlewis42/honeyssh/third_party/grep"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

// Grep is the POSIX grep command.
//
// https://pubs.opengroup.org/onlinepubs/9699919799.2018edition/utilities/grep.html
func Grep(virtOS vos.VOS) int {
	fmt.Println("starting grep")

	// Choose the context to use for function calls.
	ctx := context.Background()

	// Create a new WebAssembly Runtime.
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx) // This closes everything this Runtime created.

	// Combine the above into our baseline config, overriding defaults.
	config := wazero.NewModuleConfig().
		WithStdout(virtOS.Stdout()).
		WithStderr(virtOS.Stderr()).
		WithStdin(virtOS.Stdin()).
		WithArgs(virtOS.Args()...).
		WithFS(vos.VFSToFS(virtOS))

	// Instantiate WASI, which implements system I/O such as console output.
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	// Compile the WebAssembly module using the default configuration.
	code, err := r.CompileModule(ctx, grep.GrepWASM)
	if err != nil {
		fmt.Println(err.Error())
		return 1
	}

	if _, err = r.InstantiateModule(ctx, code, config); err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Printf(err.Error())
			return int(exitErr.ExitCode())
		} else if !ok {
			fmt.Printf(err.Error())
			return 254
		}
	}

	return 0
}

var _ vos.ProcessFunc = Grep

func init() {
	addBinCmd("grep", Grep)
}
