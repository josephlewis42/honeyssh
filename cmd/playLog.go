package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core"
)

// playLogCmd represents the playLog command
var playLogCmd = &cobra.Command{
	Use:   "play",
	Short: "Play a recorded interactive session.",
	Long:  `Plays a recorded interactive session back to the current terminal.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		fd, err := os.Open(args[0])
		if err != nil {
			return err
		}

		return core.Replay(fd, cmd.OutOrStdout())
	},
}

func init() {
	rootCmd.AddCommand(playLogCmd)
}
