package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core/logger"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Explore the honeypot event log.",
}

var reportCommand = &cobra.Command{
	Use:   "report",
	Short: "Show a report of events.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		config, err := loadConfig()
		if err != nil {
			return err
		}

		fd, err := config.ReadAppLog()
		if err != nil {
			return err
		}
		defer fd.Close()

		var report logger.Report
		if err := logger.ReadJSONLinesLog(fd, report.Update); err != nil {
			return err
		}

		fmt.Printf("%#v\n", report)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(reportCommand)
}
