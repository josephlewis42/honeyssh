package cmd

import (
	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core/config"
)

// initCmd intializes the honeypot configuration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the honeypot configuration in the current directory.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		return config.Initialize(".")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
