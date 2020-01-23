package wait

import "testing"

func TestStatusString(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		name string
		in   Status
		exp  string
	}{
		{"Start", Start, "start"},
		{"Ready", Ready, "ready"},
		{"Failed", Failed, "failed"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			exp := test.exp
			obs := test.in.String()
			if obs != exp {
				t.Errorf("%v - got: %q, want: %q", test.name, obs, exp)
			}
		})
	}
}
