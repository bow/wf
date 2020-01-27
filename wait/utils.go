package wait

import (
	"net"
	"os"
	"sync"
	"syscall"
)

// statusValues are the string representation of the Status enums.
var statusValues = []string{"start", "ready", "failed"}

// Status enumerates possible waiting status.
type Status int

const (
	// Start is the status emitted at the beginning of the wait operation.
	Start Status = iota
	// Ready is the status for when the wait operation finishes successfully.
	Ready
	// Failed is the status for when the wait operation failed.
	Failed
)

// String returns the string representation of the Status enum.
func (s Status) String() string {
	return statusValues[s]
}

// shouldWait checks that a given error represents a condition in which we should still wait and
// attempt a connection or not.
// Currently this covers two broad classes of errors:
//		1) I/O timeout errors
//		2) connection refused (server not ready) errors. Note that this has only been tested on
//		   POSIX systems.
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

// merge merges an array of channels into one channel.
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
