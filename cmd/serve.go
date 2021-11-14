/*
Copyright © 2021 Joseph Lewis <joseph@josephlewis.net>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core"
)

var (
	config = core.DefaultConfig()
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the honeypot on a local port.",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		log.Println("Initializing server...")

		log.Println("Starting logger...")
		logDest := cmd.ErrOrStderr()
		if config.LogPath != "" {
			f, err := os.OpenFile(config.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			defer f.Close()
			logDest = f
		}

		honeypot, err := core.NewHoneypot(config, logDest)
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

	serveCmd.Flags().StringVar(&config.HostKeyPath, "host-key", "", "Key for the server, random if unspecified.")
	serveCmd.Flags().IntVar(&config.SSHPort, "port", 2222, "Port to open the honeypot on.")
	serveCmd.Flags().StringVar(&config.RootFsTarPath, "root-fs", "", "Tar file to use as the root filesystem, empty if unspecified.")
	serveCmd.Flags().StringVar(&config.LogPath, "log-path", "", "Path to use as a log file. Stderr if unspecified.")
}
