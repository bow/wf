// Copyright (c) 2019-2022 Wibowo Arindrarto <contact@arindrarto.dev>
// SPDX-License-Identifier: BSD-3-Clause

package cmd

import "time"

// fmtElapsedTime creates a string representation of the given message elapsed time that is more
// human-readable (max 2 digits after decimal).
func fmtElapsedTime(et time.Duration) string {
	// Sub-microsecond time needs no special formatting.
	if et < time.Microsecond {
		return et.String()
	}

	var div uint64
	switch {
	case et < time.Millisecond:
		div = uint64(10 * time.Nanosecond)
	case et < time.Second:
		div = uint64(10 * time.Microsecond)
	default:
		div = uint64(10 * time.Millisecond)
	}

	var (
		rounder = div / 2
		val     = uint64(et)
		rem     = val % div
	)
	if rem >= rounder {
		val += rounder
	}
	et = time.Duration(val / div * div)

	return et.String()
}
