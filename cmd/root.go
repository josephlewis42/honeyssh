package cmd

import (
	"github.com/spf13/cobra"
)

var cfgPath string

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
