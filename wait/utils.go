package wait

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
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

// shouldWait checks that a given error represents a condition in which we should still wait and
// attempt a connection or not.
// Currently this covers two broad classes of errors: 1) I/O timeout errors and 2) connection
// refused (server not ready) errors. Note that this has only been tested on POSIX systems.
func shouldWait(err error) bool {
	// First case: i/o timeout.
	if os.IsTimeout(err) {
		return true
	}

	// Second case: connection refused -- remote server not ready.
	if opErr, isOpErr := err.(*net.OpError); isOpErr {
		ierr := opErr.Unwrap()
		if syscallErr, isSyscallErr := ierr.(*os.SyscallError); isSyscallErr {
			iierr := syscallErr.Unwrap()

			return iierr == syscall.ECONNREFUSED
		}
	}

	return false
}

// Adapted from: https://blog.golang.org/pipelines
func merge(chs []<-chan *TCPMessage) <-chan *TCPMessage {
	var wg sync.WaitGroup
	merged := make(chan *TCPMessage)

	forward := func(ch <-chan *TCPMessage) {
		for msg := range ch {
			merged <- msg
		}
		wg.Done()
	}

	wg.Add(len(chs))
	for _, ch := range chs {
		go forward(ch)
	}

	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}
