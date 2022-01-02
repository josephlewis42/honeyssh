package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/josephlewis42/honeyssh/core/ttylog"
	"github.com/spf13/cobra"
)

var (
	fixKippoQuirks bool
	idleTimeLimit  time.Duration
)

var logsCmd = &cobra.Command{
	Use:     "logs",
	Aliases: []string{"log"},
	Short:   "Explore the honeypot interaction logs.",
}

// playCommand represents the playLog command
var playCommand = &cobra.Command{
	Use:   "play",
	Short: "Replay a recorded interactive session in the terminal.",
	Long:  `Plays a recorded interactive session back to the current terminal.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}
		source := createLogSource(args[0], fd)

		sink := ttylog.NewClientOutput(cmd.OutOrStdout())
		sink = ttylog.NewRealTimePlayback(idleTimeLimit, sink)
		return ttylog.Replay(source, applyMiddleware(sink))
	},
}

// catCommand represents the playLog command
var catCommand = &cobra.Command{
	Use:   "cat",
	Short: "Print full output of recorded log to a terminal.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}

		source := createLogSource(args[0], fd)
		sink := ttylog.NewClientOutput(cmd.OutOrStdout())

		return ttylog.Replay(source, applyMiddleware(sink))
	},
}

// asciicastCmd converts a log to the asciicast format
var asciicastCmd = &cobra.Command{
	Use:   "asciicast INPUT.log > OUTPUT.cast",
	Short: "Convert a log to asciicast (asciinema) format.",
	Long:  `Convert a recorded terminal log to asciicast (asciinema) format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer fd.Close()

		source := ttylog.NewUMLLogSource(fd)
		sink := ttylog.NewAsciicastLogSink(cmd.OutOrStdout())

		return ttylog.Replay(source, applyMiddleware(sink))
	},
}

func createLogSource(name string, r io.Reader) ttylog.LogSource {
	switch strings.TrimPrefix(filepath.Ext(name), ".") {
	case ttylog.AsciicastFileExt:
		return ttylog.NewAsciicastLogSource(r)
	default:
		return ttylog.NewUMLLogSource(r)
	}
}

func applyMiddleware(sink ttylog.LogSink) ttylog.LogSink {
	if fixKippoQuirks {
		sink = ttylog.NewKippoQuirksAdapter(sink)
	}

	return sink
}

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.AddCommand(playCommand)
	logsCmd.AddCommand(asciicastCmd)
	logsCmd.AddCommand(catCommand)

	for _, cmd := range []*cobra.Command{playCommand, asciicastCmd, catCommand} {
		cmd.Flags().BoolVar(&fixKippoQuirks, "fix-kippo", false, "Apply fixes to logs produced by Kippo.")
	}

	// cat doesn't allow idle time
	for _, cmd := range []*cobra.Command{playCommand} {
		cmd.Flags().DurationVarP(&idleTimeLimit, "idle-time-limit", "i", 3*time.Second, "Maximum time output can be idle. (e.g. 3s, 2m, 100ms)")
	}
}
