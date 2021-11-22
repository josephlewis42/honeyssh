package cmd

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/commands"
)

// serveCmd represents the serve command
var builtinsCmd = &cobra.Command{
	Use:   "builtins",
	Short: "Show the builtin commands for the honeypot.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var builtins []string

		for path, _ := range commands.AllCommands {
			builtins = append(builtins, path)
		}

		for cmd, _ := range commands.AllBuiltins {
			builtins = append(builtins, "shell:"+cmd)
		}

		sort.Strings(builtins)

		for _, v := range builtins {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(builtinsCmd)
}
