package wait

import (
	"fmt"
	"sync"
)

// WaitStatus enumerates possible waiting status.
type WaitStatus int

const (
	Waiting WaitStatus = iota
	Ready
	Failed
)

func (ws WaitStatus) String() string {
	return [...]string{"waiting", "ready", "failed"}[ws]
}

// pendingSet is a set container for addresses to which a TCP connection has not been made.
type pendingSet struct {
	members map[string]bool
	mux     sync.Mutex
}

// newPendingSet creates a set containing the given addresses.
func newPendingSet(addrs []string) *pendingSet {
	members := make(map[string]bool, len(addrs))
	for _, addr := range addrs {
		members[addr] = true
	}

	return &pendingSet{members: members}
}

// Remove removes the given address from the set. It is safe to use concurrently. The given address
// may or may not exist prior to removal.
func (ps *pendingSet) Remove(addr string) {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	delete(ps.members, addr)
}

/// IsEmpty checks whether the set is empty or not. It is safe to use concurrently.
func (ps *pendingSet) IsEmpty() bool {
	ps.mux.Lock()
	defer ps.mux.Unlock()

	return len(ps.members) == 0
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
