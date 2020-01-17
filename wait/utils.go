package wait

import (
	"fmt"
)

var statusValues = []string{"waiting", "ready", "failed"}

// Status enumerates possible waiting status.
type Status int

const (
	Waiting Status = iota
	Ready
	Failed
)

func (s Status) String() string {
	return statusValues[s]
}

// maxLength calculates the maximum length of the given strings.
func maxLength(values []string) int {
	var result int

	for _, value := range values {
		if curLen := len(value); curLen > result {
			result = curLen
		}
	}

	return result
}

// mkFmtVerb creates a format verb with a the proper spacing and padding suitable for all the given
// string values.
func mkFmtVerb(values []string, padding int, leftJustify bool) string {
	ml := maxLength(values)

	multiplier := -1
	if !leftJustify {
		multiplier = 1
	}

	verb := fmt.Sprintf("%%%ds", multiplier*(ml+padding))

	return verb
}
