package cmd

import (
	"errors"
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
				return errors.New("at least one address must be specified")
			}
			return nil
		},

		Run: func(cmd *cobra.Command, args []string) {
			var addrs []string
			dashIdx := cmd.ArgsLenAtDash()
			if dashIdx == -1 {
				addrs = args
			} else {
				addrs = args[:dashIdx]
			}

			// TODO: Update raw address syntax to set this.
			checkFreq := 300 * time.Millisecond
			replyTimeout := 500 * time.Millisecond

			cfgs := make([]*wait.TCPInputConfig, len(addrs))
			for i, addr := range addrs {
				cfgs[i] = &wait.TCPInputConfig{
					Addr:         addr,
					CheckFreq:    checkFreq,
					ReplyTimeout: replyTimeout,
				}
			}

			msg := wait.WaitAllTCP(cfgs, waitTimeout, statusFreq, isQuiet)
			if msg.Err != nil {
				if !isQuiet {
					fmt.Printf("ERROR: %s\n", msg.Err)
				}
				os.Exit(1)
			}
			if !isQuiet {
				fmt.Printf("OK: all ready after %s\n", msg.SinceStart())
			}
		},
	}

	cmd.Flags().DurationVarP(&waitTimeout, "timeout", "t", 5*time.Second, "set wait timeout")
	cmd.Flags().DurationVarP(&statusFreq, "status-freq", "s", 1*time.Second, "set status message frequency")
	cmd.Flags().BoolVarP(&isQuiet, "quiet", "q", false, "suppress waiting messages")

	return cmd.Execute()
}
