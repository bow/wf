// Copyright (c) 2019-2022 Wibowo Arindrarto <contact@arindrarto.dev>
// SPDX-License-Identifier: BSD-3-Clause

package cmd

import (
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	t.Parallel()

	retCode := run([]string{"golang.org:443"}, 5*time.Second, 500*time.Millisecond, false)

	if retCode != 0 {
		t.Errorf("test failed - want exit code: %d, got: %d", 0, retCode)
	}
}
