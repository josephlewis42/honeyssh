package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/josephlewis42/honeyssh/core/logger"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	eventsFilter func(*logger.LogEntry) bool
	sinceTime    *string
	since        *time.Duration
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Explore the honeypot event log.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch {
		case *since > 0 && *sinceTime != "":
			return errors.New("can't supply both since and since-time")
		case *since > 0:
			sinceMicros := time.Now().UnixMicro() - since.Microseconds()
			eventsFilter = func(le *logger.LogEntry) bool {
				return le.TimestampMicros >= sinceMicros
			}
		case *sinceTime != "":
			parsedSinceTime, err := time.Parse(time.RFC3339, *sinceTime)
			if err != nil {
				return fmt.Errorf("couldn't parse since-time: %v", err)
			}
			eventsFilter = func(le *logger.LogEntry) bool {
				return le.TimestampMicros >= parsedSinceTime.UnixMicro()
			}
		default:
			eventsFilter = func(*logger.LogEntry) bool {
				return true
			}
		}

		return nil
	},
}

var summaryCommand = &cobra.Command{
	Use:   "summary",
	Short: "Show a summary of events.",
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
		if err := logger.ReadJSONLinesLog(fd, func(le *logger.LogEntry) {
			if eventsFilter(le) {
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

var interactionsCommand = &cobra.Command{
	Use:   "interactions",
	Short: "Show a summary of interactions.",
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

		report := &logger.InteractionReport{}
		if err := logger.ReadJSONLinesLog(fd, func(le *logger.LogEntry) {
			if eventsFilter(le) {
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
	eventsCmd.AddCommand(summaryCommand)
	eventsCmd.AddCommand(interactionsCommand)

	since = eventsCmd.Flags().Duration("since", -1, "Display events newer than a relative duration. e.g. 24h")
	sinceTime = eventsCmd.Flags().String("since-time", "", "Display events after a specific date (RFC3339).")
}
