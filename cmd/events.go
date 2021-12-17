package cmd

import (
	"github.com/spf13/cobra"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Explore the honeypot event log.",
}

func init() {
	rootCmd.AddCommand(eventsCmd)
}
