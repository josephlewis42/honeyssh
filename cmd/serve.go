package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"josephlewis.net/honeyssh/core"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the honeypot on a local port.",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		os.Stdin.Close()
		cmd.SilenceUsage = true
		log.Println("Initializing server...")

		log.Println("Starting logger...")
		logDest := cmd.ErrOrStderr()

		configuration, err := loadConfig()
		if err != nil {
			return err
		}

		honeypot, err := core.NewHoneypot(configuration, logDest)
		if err != nil {
			return err
		}

		go func() {
			if err := honeypot.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}()

		sigs := make(chan os.Signal, 1)

		log.Println("- Starting interrupt handler")
		signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		sig := <-sigs
		log.Printf("Got signal %q, terminating...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := honeypot.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed: %s", err)
		}
		log.Print("Server exited")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
