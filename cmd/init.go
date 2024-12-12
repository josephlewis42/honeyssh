package cmd

import (
	"log"

	"github.com/josephlewis42/honeyssh/core/config"
	"github.com/spf13/cobra"
)

// initCmd intializes the honeypot configuration
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the honeypot configuration in the current directory.",
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		logger := log.New(cmd.ErrOrStderr(), "", 0)

		_, err := config.Initialize(".", logger)
		return err
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
