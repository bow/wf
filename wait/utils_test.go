package wait

import "testing"

func TestStatusString(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name string
		in   Status
		want string
	}{
		{"Start", Start, "start"},
		{"Ready", Ready, "ready"},
		{"Failed", Failed, "failed"},
	}

	for i, test := range tests {
		i := i
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			name := test.name
			want := test.want
			got := test.in.String()

			if want != got {
				t.Errorf("test[%d] %q failed - want: %q, got: %q", i, name, want, got)
			}
		})
	}
}
