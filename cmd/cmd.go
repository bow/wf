package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bow/wait-for/wait"
)

const (
	appName        = "wait-for"
	appVersion     = "0.0.0"
	appDescription = "Wait until TCP server(s) are ready to accept connections"
)

// Execute peforms the actual CLI argument parsing and launches the wait operation.
func Execute() error {
	var (
		waitTimeout time.Duration
		pollFreq    time.Duration
		isQuiet     bool
	)

	cmd := &cobra.Command{
		Use:                   appName + " [FLAGS] ADDRESS...",
		Short:                 appDescription,
		Version:               appVersion,
		DisableFlagsInUseLine: true,

		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("at least one address must be specified")
			}
			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			var rawAddrs []string
			if dashIdx := cmd.ArgsLenAtDash(); dashIdx == -1 {
				rawAddrs = args
			} else {
				rawAddrs = args[:dashIdx]
			}

			specs, err := wait.ParseTCPSpecs(rawAddrs, pollFreq)
			if err != nil {
				fmt.Printf("%7s: %s\n", "ERROR", err)
				os.Exit(1)
			}

			var (
				msg   wait.Message
				showF = func(wait.Message) {}
			)
			if !isQuiet {
				showF = func(msg wait.Message) {
					var disp string

					switch msg.Status() {
					case wait.Start:
						disp = fmt.Sprintf("%7s: %s for %s", "waiting", msg.Target(), waitTimeout)
					case wait.Ready:
						disp = fmt.Sprintf("%7s: %s in %s", wait.Ready, msg.Target(), msg.ElapsedTime())
					case wait.Failed:
						disp = fmt.Sprintf("%7s: %s", wait.Failed, msg.Err())
					}

					fmt.Println(disp)
				}
			}
			for msg = range wait.AllTCP(specs, waitTimeout) {
				showF(msg)
				if err := msg.Err(); err != nil {
					os.Exit(1)
				}
			}
			// nolint:errcheck
			if !isQuiet {
				fmt.Printf("%7s: all ready in %s\n", "OK", msg.ElapsedTime())
			}
		},
	}

	flagSet := cmd.Flags()
	flagSet.SortFlags = false
	flagSet.DurationVarP(&waitTimeout, "timeout", "t", 5*time.Second, "set wait timeout")
	flagSet.DurationVarP(&pollFreq, "poll-freq", "f", 500*time.Millisecond, "set connection poll frequency")
	flagSet.BoolVarP(&isQuiet, "quiet", "q", false, "suppress waiting messages")

	return cmd.Execute()
}
