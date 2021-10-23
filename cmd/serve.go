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
	"crypto/subtle"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"

	"github.com/gliderlabs/ssh"
	"github.com/spf13/cobra"
	"josephlewis.net/osshit/core"
)

type sshContextKey struct {
	name string
}

var (
	// ContextAuthPublicKey holds the public key that the client sent to the
	// server. Useful for fingerprinting.
	ContextAuthPublicKey = sshContextKey{"auth-public-key"}
	// ContextAuthPassword holds the password the client sent to the server.
	ContextAuthPassword = sshContextKey{"auth-password"}

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
		server := &ssh.Server{
			Addr: fmt.Sprintf(":%d", config.SSHPort),
			Handler: func(s ssh.Session) {
				// Duplicate output for things like scp or git which don't.
				out := os.Stdout

				tw := tabwriter.NewWriter(out, 8, 4, 4, ' ', 0)
				fmt.Fprintf(tw, "Username\t%s\n", s.User())
				fmt.Fprintf(tw, "Public Key\t%q\n", s.Context().Value(ContextAuthPublicKey))
				fmt.Fprintf(tw, "Password\t%q\n", s.Context().Value(ContextAuthPassword))
				fmt.Fprintf(tw, "RemoteAddr\t%q\n", s.RemoteAddr())
				fmt.Fprintf(tw, "Environ\t%q\n", s.Environ())
				fmt.Fprintf(tw, "Command\t%q\n", s.Command())
				fmt.Fprintf(tw, "RawCommand\t%q\n", s.RawCommand())
				fmt.Fprintf(tw, "Subsystem\t%q\n", s.Subsystem())

				tw.Flush()

				fakeShell, err := core.NewShell(s, config)
				if err != nil {
					log.Println(err)
					s.Exit(1)
					return
				}

				// run shell
				fakeShell.Run()
				fakeShell.Close()

				s.Exit(0)
			},
			PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
				ctx.SetValue(ContextAuthPublicKey, key.Marshal())
				return false
			},
			PasswordHandler: func(ctx ssh.Context, password string) bool {
				ctx.SetValue(ContextAuthPassword, password)
				return 0 == subtle.ConstantTimeCompare([]byte(password), []byte("password"))
			},
		}

		if config.HostKeyPath != "" {
			log.Printf("- Using host key: %q\n", config.HostKeyPath)
			server.SetOption(ssh.HostKeyFile(config.HostKeyPath))
		}

		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		serverErr := make(chan error, 1)

		go func() {
			log.Printf("- Starting SSH server on %s\n", server.Addr)
			serverErr <- server.ListenAndServe()
		}()

		log.Println("- Starting interrupt handler")
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
		go func() {
			for {
				sig := <-sigs
				switch sig {
				case syscall.SIGINT:
					// TODO send out a maintenance signal.
					fallthrough
				case syscall.SIGTERM, syscall.SIGKILL:
					log.Printf("Got signal %q, terminating...", sig)
					done <- true
					return
				}
			}
		}()

		select {
		case err := <-serverErr:
			// Failure
			return fmt.Errorf("server failure: %v", err)

		case <-done:
			// graceful termination
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().StringVar(&config.HostKeyPath, "host-key", "", "Key for the server, random if unspecified.")
	serveCmd.Flags().IntVar(&config.SSHPort, "port", 2222, "Port to open the honeypot on.")
	serveCmd.Flags().StringVar(&config.RootFsTarPath, "root-fs", "", "Tar file to use as the root filesystem, empty if unspecified.")
}
