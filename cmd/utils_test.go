package cmd

import (
	"testing"
	"time"
)

func TestFmtElapsedTime(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		in   time.Duration
		want string
	}{
		{0 * time.Nanosecond, "0s"},
		{45 * time.Nanosecond, "45ns"},
		{24313 * time.Nanosecond, "24.31µs"},
		{759825 * time.Nanosecond, "759.83µs"},
		{999995 * time.Nanosecond, "1ms"},
		{999994 * time.Nanosecond, "999.99µs"},
		{32423 * time.Microsecond, "32.42ms"},
		{301451654 * time.Microsecond, "5m1.45s"},
		{287336 * time.Millisecond, "4m47.34s"},
		{125432 * time.Millisecond, "2m5.43s"},
		{301 * time.Second, "5m1s"},
	}

	for i, test := range tests {
		i := i
		test := test
		name := test.in.String()

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			want := test.want
			got := fmtElapsedTime(test.in)

			if want != got {
				t.Errorf("test[%d] %q failed - want: %q, got: %q", i, name, want, got)
			}
		})
	}
}
