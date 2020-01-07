package wait

import (
	"fmt"
	"testing"
)

func TestStatusString(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		name string
		in   Status
		exp  string
	}{
		{"Waiting", Waiting, "waiting"},
		{"Ready", Ready, "ready"},
		{"Failed", Failed, "failed"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			exp := test.exp
			obs := fmt.Sprintf("%s", test.in)
			if obs != exp {
				t.Errorf("%v - got: %q, want: %q", test.name, obs, exp)
			}
		})
	}
}

func TestPendingSet(t *testing.T) {
	t.Parallel()

	set := newPendingSet([]string{"a", "b"})

	t.Run("IsEmpty() on a nonempty set works", func(t *testing.T) {
		if set.IsEmpty() {
			t.Errorf("expected set to have two items")
		}
	})

	t.Run("Remove() with nonexistent item works", func(t *testing.T) {
		set.Remove("c")
		if set.IsEmpty() {
			t.Errorf("expected set to still have two items")
		}
	})

	t.Run("Remove() with existing item down to 1 members works", func(t *testing.T) {
		set.Remove("a")
		if set.IsEmpty() {
			t.Errorf("expected set to still have one item")
		}
	})

	t.Run("Remove() with existing item down to 0 members works", func(t *testing.T) {
		set.Remove("b")
		if !set.IsEmpty() {
			t.Errorf("expected set to be empty")
		}
	})
}

func TestMaxLength(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		name string
		in   []string
		exp  int
	}{
		{"empty", []string{}, 0},
		{"first item is max", []string{"aaa", "aa", "a"}, 3},
		{"last item is max", []string{"a", "aa", "aaa"}, 3},
		{"multiple items are max", []string{"aaa", "aa", "aaa"}, 3},
	}

	for i, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			exp := test.exp
			obs := maxLength(test.in)
			if obs != exp {
				t.Errorf("test[%d] %q failed - got: %d, want: %d", i, test.name, obs, exp)
			}
		})
	}
}

func TestMkFmtVerb(t *testing.T) {
	t.Parallel()
	var tests = []struct {
		name          string
		inValues      []string
		inPadding     int
		inLeftJustify bool
		exp           string
	}{
		{"pad < maxlen; left justify", []string{"a", "ccc", "a"}, 0, true, "%-3s"},
		{"pad < maxlen; right justify", []string{"a", "ccc", "a"}, 0, false, "%3s"},
		{"pad > maxlen; left justify", []string{"a", "ccc", "a"}, 5, true, "%-8s"},
		{"pad > maxlen; right justify", []string{"a", "ccc", "a"}, 5, false, "%8s"},
	}

	for i, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			exp := test.exp
			obs := mkFmtVerb(test.inValues, test.inPadding, test.inLeftJustify)
			if obs != exp {
				t.Errorf("test[%d] %q failed - got: %q, want: %q", i, test.name, obs, exp)
			}
		})
	}
}
