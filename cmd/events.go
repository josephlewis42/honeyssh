package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core/logger"
	"sigs.k8s.io/yaml"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Explore the honeypot event log.",
}

var (
	sinceTime *string
	since     *time.Duration
)

var reportCommand = &cobra.Command{
	Use:   "report",
	Short: "Show a report of events.",
	RunE: func(cmd *cobra.Command, args []string) error {

		var filter func(*logger.LogEntry) bool
		switch {
		case *since > 0 && *sinceTime != "":
			return errors.New("can't supply both since and since-time")
		case *since > 0:
			sinceMicros := time.Now().UnixMicro() - since.Microseconds()
			filter = func(le *logger.LogEntry) bool {
				return le.TimestampMicros >= sinceMicros
			}
		case *sinceTime != "":
			parsedSinceTime, err := time.Parse(time.RFC3339, *sinceTime)
			if err != nil {
				return fmt.Errorf("couldn't parse since-time: %v", err)
			}
			filter = func(le *logger.LogEntry) bool {
				return le.TimestampMicros >= parsedSinceTime.UnixMicro()
			}
		default:
			filter = func(*logger.LogEntry) bool {
				return true
			}
		}

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
		if err := logger.ReadJSONLinesLog(fd, func(le *logger.LogEntry) {
			if filter(le) {
				report.Update(le)
			}
		}); err != nil {
			return err
		}

		out, err := yaml.Marshal(report)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), string(out))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(eventsCmd)
	eventsCmd.AddCommand(reportCommand)

	since = reportCommand.Flags().Duration("since", -1, "Display events newer than a relative duration. e.g. 24h")
	sinceTime = reportCommand.Flags().String("since-time", "", "Display events after a specific date (RFC3339).")
}
