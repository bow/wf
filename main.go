// Package main exposes the https://godoc.org/github.com/bow/wait-for/wait package as the wait-for
// command line application.
//
// The driving use case for wait-for is to make containerized applications that depend on external
// services more robust by waiting for those actual services to be ready, prior to application
// start. It is provided as a single, static binary for linux-amd64, so it can be used as-is.
package main

import (
	"fmt"
	"os"

	"github.com/bow/wait-for/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
