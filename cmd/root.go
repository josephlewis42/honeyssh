package cmd

import (
	"errors"
	"io/fs"
	"log"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core/config"
)

var cfgPath string

func loadConfig() (*config.Configuration, error) {
	configuration, err := config.Load(cfgPath)

	if errors.Is(err, fs.ErrNotExist) {
		log.Println("Couldn't load config: did you run init?")
	}

	return configuration, err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "osshit",
	Short: "Open SSH Intrusion Tracker",
	Long:  `A medium interaction SSH honeypot.`,
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", ".", "config path")
}
