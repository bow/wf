package main

import (
	"fmt"
	"os"
	"time"

	. "github.com/bow/wait-for/wait"
)

func main() {

	// TODO: Make these variables configurable via CLI.
	isQuiet := false
	waitTimeout := 3 * time.Second
	checkFreq := 300 * time.Millisecond
	statusFreq := 500 * time.Millisecond
	replyTimeout := 500 * time.Millisecond
	cfgs := []*TCPInputConfig{
		&TCPInputConfig{
			Addr:         "localhost:8000",
			CheckFreq:    checkFreq,
			ReplyTimeout: replyTimeout,
		},
		&TCPInputConfig{
			Addr:         "localhost:5672",
			CheckFreq:    checkFreq,
			ReplyTimeout: replyTimeout,
		},
		&TCPInputConfig{
			Addr:         "google.com:80",
			CheckFreq:    checkFreq,
			ReplyTimeout: replyTimeout,
		},
	}

	msg := WaitAllTCP(cfgs, waitTimeout, statusFreq, isQuiet)
	if msg.Err != nil {
		if !isQuiet {
			fmt.Printf("ERROR: %s\n", msg.Err)
		}
		os.Exit(1)
	}
	if !isQuiet {
		fmt.Printf("OK: all ready after %s\n", msg.SinceStart())
	}
}
