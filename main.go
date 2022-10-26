// Copyright (c) 2019-2022 Wibowo Arindrarto <contact@arindrarto.dev>
// SPDX-License-Identifier: BSD-3-Clause

// Package main exposes the https://godoc.org/github.com/bow/wf/wait package as the wf command line
// application.
//
// The main use case for wf is to make containerized applications that depend on external services
// more robust by waiting for those actual services to be ready, prior to application start. It is
// provided as a single, static binary for linux-amd64, so it can be added into the container and
// used as-is.
package main

import (
	"fmt"
	"os"

	"github.com/bow/wf/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
