// Package cmd provides the command line interface for the
// https://godoc.org/github.com/bow/wait-for/wait package.
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

// Execute peforms the actual command line argument parsing and launches the wait operation.
func Execute() error {
	var (
		waitTimeout     time.Duration
		defaultPollFreq time.Duration
		isQuiet         bool
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
			exitCode := run(cmd, args, waitTimeout, defaultPollFreq, isQuiet)
			if exitCode != 0 {
				os.Exit(exitCode)
			}
		},
	}

	flagSet := cmd.Flags()
	flagSet.SortFlags = false
	flagSet.DurationVarP(&waitTimeout, "timeout", "t", 5*time.Second, "set wait timeout")
	flagSet.DurationVarP(
		&defaultPollFreq,
		"poll-freq",
		"f",
		500*time.Millisecond,
		"set connection poll frequency",
	)
	flagSet.BoolVarP(&isQuiet, "quiet", "q", false, "suppress waiting messages")

	return cmd.Execute()
}

// run calls the actual function for waiting.
func run(
	cmd *cobra.Command,
	args []string,
	waitTimeout, defaultPollFreq time.Duration,
	isQuiet bool,
) int {
	var rawAddrs []string
	if dashIdx := cmd.ArgsLenAtDash(); dashIdx == -1 {
		rawAddrs = args
	} else {
		rawAddrs = args[:dashIdx]
	}

	specs, err := wait.ParseTCPSpecs(rawAddrs, defaultPollFreq)
	if err != nil {
		fmt.Printf("%7s: %s\n", "ERROR", err)
		return 1
	}

	var (
		msg       wait.Message
		showMsg   = func(wait.Message) {}
		showFinal = func(wait.Message) {}
	)
	if !isQuiet {
		showMsg = func(msg wait.Message) {
			var disp string

			switch msg.Status() {
			case wait.Start:
				disp = fmt.Sprintf("%7s: %s for %s", "waiting", msg.Target(), waitTimeout)
			case wait.Ready:
				disp = fmt.Sprintf(
					"%7s: %s in %s",
					wait.Ready,
					msg.Target(),
					msg.ElapsedTime(),
				)
			case wait.Failed:
				disp = fmt.Sprintf("%7s: %s", wait.Failed, msg.Err())
			}

			fmt.Println(disp)
		}
		showFinal = func(msg wait.Message) {
			fmt.Printf("%7s: all ready in %s\n", "OK", msg.ElapsedTime())
		}
	}

	for msg = range wait.AllTCP(specs, waitTimeout) {
		showMsg(msg)
		if err := msg.Err(); err != nil {
			return 1
		}
	}
	showFinal(msg)

	return 0
}
