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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// playLogCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// playLogCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
