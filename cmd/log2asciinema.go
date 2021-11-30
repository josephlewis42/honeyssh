package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core"
)

var (
	crlf = regexp.MustCompile(`\r?\n`)
)

// log2Asciicast converts a log to the asciicast format
var log2Asciicast = &cobra.Command{
	Use:   "log2asciicast INPUT > OUTPUT.cast",
	Short: "Convert a log to asciicast format.",
	Long:  `Convert a recorded terminal log to asciicast format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer fd.Close()

		writeJSONLine := func(structure interface{}) error {
			line, err := json.Marshal(structure)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", string(line))
			return err
		}

		// Write header line
		if err := writeJSONLine(map[string]interface{}{
			"version":   2,
			"width":     80,
			"height":    24,
			"timestamp": 0,
			"title":     filepath.Base(args[0]),
			"env": map[string]interface{}{
				"TERM":  "xterm-256color",
				"SHELL": "/bin/bash",
			},
		}); err != nil {
			return err
		}

		var startTime time.Time
		var once sync.Once
		var skew float64
		var lastTimeSinceStart float64

		return core.ReplayCallback(fd, func(event *core.LogEvent) error {
			once.Do(func() {
				startTime = event.Time
			})

			timeSinceStartSeconds := float64(event.Time.Sub(startTime)) / float64(time.Second)

			// max pause of 3 seconds
			if pause := timeSinceStartSeconds - lastTimeSinceStart; pause > 3.0 {
				skew += -pause + 3.0
			}

			lastTimeSinceStart = timeSinceStartSeconds
			timeSinceStartSeconds += skew

			eventType := ""
			switch event.EventType {
			case core.EventTypeInput:
				eventType = "i"
			case core.EventTypeOutput:
				eventType = "o"
			default:
				// Some other event, don't care.
				return nil
			}

			replaced := crlf.ReplaceAllString(string(event.Data), "\r\n")
			line := []interface{}{timeSinceStartSeconds, eventType, replaced}
			return writeJSONLine(line)
		})
	},
}

func init() {
	rootCmd.AddCommand(log2Asciicast)
}
