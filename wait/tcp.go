package wait

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"time"
)

// TCPWaitMessage is a container for a wait operation result.
type TCPWaitMessage struct {
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
func (msg *TCPWaitMessage) SinceStart() time.Duration {
	return msg.EmitTime.Sub(msg.StartTime)
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
	chWaiting chan TCPWaitMessage,
	chReady chan TCPWaitMessage,
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
			chReady <- TCPWaitMessage{Ready, addr, startTime, time.Now(), nil}
			return false
		}

		if ShouldWait(err) {
			return true
		} else {
			chReady <- TCPWaitMessage{Failed, addr, startTime, time.Now(), err}
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
			chWaiting <- TCPWaitMessage{Waiting, addr, startTime, time.Now(), nil}
		}
	}
}

// TCPInputConfig represents the inputs to a single wait operation for TCP addresses.
type TCPInputConfig struct {
	// Addr is the address being waited.
	Addr string
	// CheckFreq is how often a connection is attempted.
	CheckFreq time.Duration
	// ReplyTimeout is how long to wait for a reply from the server before erroring out.
	ReplyTimeout time.Duration
}

// WaitAllTCP waits until connections can be made to all given TCP addresses.
func WaitAllTCP(
	configs []*TCPInputConfig,
	waitTimeout time.Duration,
	statusFreq time.Duration,
	isQuiet bool,
) TCPWaitMessage {

	// Initialize a slice of addresses; used for initializing a pending set and determining padding
	// when printing.
	addrs := make([]string, len(configs))
	for i, config := range configs {
		addrs[i] = config.Addr
	}

	var (
		showStatus         func(TCPWaitMessage)
		pendingSet         = newPendingSet(addrs)
		ready              = make(chan TCPWaitMessage)
		waiting            = make(chan TCPWaitMessage)
		startTime, timeout = time.Now(), time.NewTimer(waitTimeout)
	)
	defer timeout.Stop()

	if isQuiet {
		// Set status freq to be double the wait timeout, preventing any status from being emitted.
		statusFreq = 2 * waitTimeout
		// Set showStatus to do nothing; needed even without any status being emitted to prevent
		// runtime nil deref runtime error.
		showStatus = func(msg TCPWaitMessage) {}
	} else {
		statusVerb := mkFmtVerb([]string{Ready.String(), Waiting.String()}, 0, false)
		addrVerb := mkFmtVerb(addrs, 2, true)
		msgFmt := fmt.Sprintf("%s: %s [%%s] ...\n", statusVerb, addrVerb)
		showStatus = func(msg TCPWaitMessage) {
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
			return TCPWaitMessage{
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
