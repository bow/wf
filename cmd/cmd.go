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
	appDescription = "Launch process when TCP server(s) are ready"
)

func Execute() error {
	var (
		waitTimeout time.Duration
		pollFreq    time.Duration
		statusFreq  time.Duration
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

			specs, err := wait.ParseTCPSpecs(rawAddrs, pollFreq, statusFreq)
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				os.Exit(1)
			}

			var msg wait.Message
			for msg = range wait.AllTCP(specs, waitTimeout) {
				if !isQuiet {
					fmt.Println(msg.String())
				}
				if err := msg.Err(); err != nil {
					os.Exit(1)
				}
			}
			// nolint:errcheck
			if !isQuiet {
				fmt.Printf("Ok: all ready after %s\n", msg.ElapsedTime())
			}
		},
	}

	flagSet := cmd.Flags()
	flagSet.SortFlags = false
	flagSet.DurationVarP(&waitTimeout, "timeout", "t", 5*time.Second, "set wait timeout")
	flagSet.DurationVarP(&pollFreq, "poll-freq", "f", 500*time.Millisecond, "set connection poll frequency")
	flagSet.DurationVarP(&statusFreq, "status-freq", "s", 1*time.Second, "set status message frequency")
	flagSet.BoolVarP(&isQuiet, "quiet", "q", false, "suppress waiting messages")

	return cmd.Execute()
}
