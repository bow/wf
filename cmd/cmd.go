// Package cmd provides the command line interface for the
// https://godoc.org/github.com/bow/wf/wait package.
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/bow/wf/wait"
)

const (
	name = "wf"
	desc = "Wait until TCP server(s) are ready to accept connections"
)

var (
	// These are meant to be overidden at built time using ldflags -X.
	buildTime = "?"
	version   = "dev"
	gitCommit = "?"
)

// Execute peforms the actual CLI argument parsing and launches the wait operation.
func Execute() error {
	var (
		waitTimeout     time.Duration
		defaultPollFreq time.Duration
		isQuiet         bool

		ver = fmt.Sprintf("%s (build time: %s, commit: %s)", version, buildTime, gitCommit)
	)

	cmd := &cobra.Command{
		Use:                   name + " [FLAGS] ADDRESS...",
		Short:                 desc,
		Version:               ver,
		DisableFlagsInUseLine: true,
		SilenceErrors:         true,

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
			exitCode := run(rawAddrs, waitTimeout, defaultPollFreq, isQuiet)
			if exitCode != 0 {
				os.Exit(exitCode) // nolint: revive
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
	rawAddrs []string,
	waitTimeout, defaultPollFreq time.Duration,
	isQuiet bool,
) int {

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
					fmtElapsedTime(msg.ElapsedTime()),
				)
			case wait.Failed:
				disp = fmt.Sprintf("%7s: %s", wait.Failed, msg.Err())
			}

			fmt.Println(disp)
		}
		showFinal = func(msg wait.Message) {
			fmt.Printf("%7s: all ready in %s\n", "OK", fmtElapsedTime(msg.ElapsedTime()))
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
