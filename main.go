package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
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

// WaitMessage is a container for a wait operation result.
type WaitMessage struct {
	// Status is the status of the waiting operation.
	Status WaitStatus
	// Addr is the address being waited.
	Addr string
	// Start is the start time of the wait operation.
	StartTime time.Time
	// Time is the time the status was emitted.
	EmitTime time.Time
	// Err is any possible errors that have occured.
	Err error
}

// SinceStart returns the duration between status emission and wait start time.
func (msg *WaitMessage) SinceStart() time.Duration {
	return msg.EmitTime.Sub(msg.StartTime)
}

// InputConfig represents the inputs to a single wait operation.
type InputConfig struct {
	// Addr is the address being waited.
	Addr string
	// CheckFreq is how often a connection is attempted.
	CheckFreq time.Duration
	// ReplyTimeout is how long to wait for a reply from the server before erroring out.
	ReplyTimeout time.Duration
}

// ShouldWait checks that a given error represents a condition in which we should still wait and
// attempt a connection or not.
// Currently this covers two broad classes of errors: 1) I/O timeout errors and 2) connection
// refused (server not ready) errors. Note that this only been tested on POSIX systems.
func ShouldWait(err error) bool {
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

// WaitSingleTCP waits until a TCP connection can be made to the given address. It runs
// indefinitely, emitting messages to two channels: `chWaiting` while still waiting and `chReady`
// when the wait has finished. The check frequency is controlled by `checkFreq`, while every
// `statusFreq` a status message is emitted. The timeout for the server reply is determined by
// `replyTimeout`. A `startTime` may be given a nonzero value, which is useful when tracking
// multiple wait operations launched within a short period. If its value is equal to the zero time,
// the current time is used.
func WaitSingleTCP(
	addr string,
	chWaiting chan WaitMessage,
	chReady chan WaitMessage,
	checkFreq, statusFreq, replyTimeout time.Duration,
	startTime time.Time,
) {

	if startTime.IsZero() {
		startTime = time.Now()
	}

	pollTicker := time.NewTicker(checkFreq)
	defer pollTicker.Stop()

	statusTicker := time.NewTicker(statusFreq)
	defer statusTicker.Stop()

	check := func() bool {
		_, err := net.DialTimeout("tcp", addr, replyTimeout)

		if err == nil {
			chReady <- WaitMessage{Ready, addr, startTime, time.Now(), nil}
			return false
		}

		if ShouldWait(err) {
			return true
		} else {
			chReady <- WaitMessage{Failed, addr, startTime, time.Now(), err}
			return false
		}
	}

	// So that we start polling immediately, without waiting for the first tick.
	// There is no way to do this via the current ticker API.
	// See: https://github.com/golang/go/issues/17601
	keepWaiting := check()
	if !keepWaiting {
		return
	}

	for {
		select {
		case <-pollTicker.C:
			keepWaiting = check()
			if !keepWaiting {
				return
			}

		case <-statusTicker.C:
			chWaiting <- WaitMessage{Waiting, addr, startTime, time.Now(), nil}
		}
	}
}

// WaitAll waits until connections can be made to all given TCP addresses.
func WaitAll(
	configs []*InputConfig,
	waitTimeout time.Duration,
	statusFreq time.Duration,
	isQuiet bool,
) WaitMessage {

	// Initialize a slice of addresses; used for initializing a pending set and determining padding
	// when printing.
	addrs := make([]string, len(configs))
	for i, config := range configs {
		addrs[i] = config.Addr
	}

	var (
		showStatus         func(WaitMessage)
		pendingSet         = newPendingSet(addrs)
		ready              = make(chan WaitMessage)
		waiting            = make(chan WaitMessage)
		startTime, timeout = time.Now(), time.NewTimer(waitTimeout)
	)
	defer timeout.Stop()

	if isQuiet {
		// Set status freq to be double the wait timeout, preventing any status from being emitted.
		statusFreq = 2 * waitTimeout
		// Set showStatus to do nothing; needed even without any status being emitted to prevent
		// runtime nil deref runtime error.
		showStatus = func(msg WaitMessage) {}
	} else {
		statusVerb := mkFmtVerb([]string{Ready.String(), Waiting.String()}, 0, false)
		addrVerb := mkFmtVerb(addrs, 2, true)
		msgFmt := fmt.Sprintf("%s: %s [%%s] ...\n", statusVerb, addrVerb)
		showStatus = func(msg WaitMessage) {
			fmt.Printf(msgFmt, msg.Status, msg.Addr, msg.SinceStart())
		}
	}

	for _, config := range configs {
		go WaitSingleTCP(
			config.Addr,
			waiting,
			ready,
			config.CheckFreq,
			statusFreq,
			config.ReplyTimeout,
			startTime,
		)
	}

	for {
		select {
		case <-timeout.C:
			return WaitMessage{
				Err: fmt.Errorf("reached timeout limit of %s", waitTimeout),
			}

		case waitMsg := <-waiting:
			showStatus(waitMsg)

		case readyMsg := <-ready:
			if readyMsg.Err != nil {
				return readyMsg
			}

			pendingSet.Remove(readyMsg.Addr)
			showStatus(readyMsg)

			if pendingSet.IsEmpty() {
				return readyMsg
			}
		}
	}
}

func main() {

	// TODO: Make these variables configurable via CLI.
	isQuiet := false
	waitTimeout := 3 * time.Second
	checkFreq := 300 * time.Millisecond
	statusFreq := 500 * time.Millisecond
	replyTimeout := 500 * time.Millisecond
	cfgs := []*InputConfig{
		&InputConfig{"localhost:8000", checkFreq, replyTimeout},
		&InputConfig{"localhost:5672", checkFreq, replyTimeout},
		&InputConfig{"google.com:80", checkFreq, replyTimeout},
	}

	msg := WaitAll(cfgs, waitTimeout, statusFreq, isQuiet)
	if msg.Err != nil {
		if !isQuiet {
			fmt.Printf("ERROR: %s\n", msg.Err)
		}
		os.Exit(1)
	}
	if !isQuiet {
		fmt.Printf("OK: all ready after %s\n", msg.SinceStart())
	}
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
