package wait

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var (
	addrPattern = regexp.MustCompile(
		"^(?P<schema>(?P<proto>[A-Za-z]+)://)?(?P<host>[^#]+)(#(?P<freq>.+))?",
	)
	protoPort = map[string]string{
		"amqp":  "5672",
		"amqps": "5671",
		"http":  "80",
		"https": "443",
		"imap":  "143",
		"mysql": "3306",
		"ldap":  "389",
		"ldaps": "636",
		"psql":  "5432",
		"smtp":  "25",
	}
)

// TCPSpec represents the input specification of a single TCP wait operation.
type TCPSpec struct {
	// Host is the hostname or IP address being waited.
	Host string
	// Port is the port number for the connection.
	Port string
	// PollFreq is how often a connection is attempted.
	PollFreq time.Duration
}

// TCPMessage is a container for a TCP wait operation status.
type TCPMessage struct {
	// Status is the status of the waiting operation.
	Status Status
	// Addr is the address being waited.
	Addr string
	// Start is the start time of the wait operation.
	StartTime time.Time
	// Time is the time the status was emitted.
	EmitTime time.Time
	// Err is any possible errors that have occurred.
	Err error
}

// SinceStart returns the duration between status emission and wait start time.
func (msg *TCPMessage) SinceStart() time.Duration {
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

// SingleTCP waits until a TCP connection can be made to the given address. It runs indefinitely,
// emitting messages to two channels: `chWaiting` while still waiting and `chReady` when the wait
// has finished. The check frequency is controlled by `pollFreq`, while every `statusFreq` a status
// message is emitted A `startTime` may be given a nonzero value, which is useful when tracking
// multiple wait operations launched within a short period. If its value is equal to the zero time,
// the current time is used.
func SingleTCP(
	addr string,
	chWaiting chan TCPMessage,
	chReady chan TCPMessage,
	pollFreq, statusFreq time.Duration,
	startTime time.Time,
) {

	if startTime.IsZero() {
		startTime = time.Now()
	}

	pollTicker := time.NewTicker(pollFreq)
	defer pollTicker.Stop()

	statusTicker := time.NewTicker(statusFreq)
	defer statusTicker.Stop()

	// Helper function to check if a connection can be established.
	checkConn := func() bool {
		_, err := net.DialTimeout("tcp", addr, pollFreq)

		if err == nil {
			chReady <- TCPMessage{Ready, addr, startTime, time.Now(), nil}
			return true
		}

		if ShouldWait(err) {
			return false
		}

		chReady <- TCPMessage{Failed, addr, startTime, time.Now(), err}
		return true
	}

	// So that we start polling immediately, without waiting for the first tick.
	// There is no way to do this via the current ticker API.
	// See: https://github.com/golang/go/issues/17601
	if connReady := checkConn(); connReady {
		return
	}

	for {
		select {
		case <-pollTicker.C:
			if connReady := checkConn(); connReady {
				return
			}

		case <-statusTicker.C:
			chWaiting <- TCPMessage{Waiting, addr, startTime, time.Now(), nil}
		}
	}
}

func ParseTCPSpec(addr string, pollFreq time.Duration) (*TCPSpec, error) {
	var (
		proto             string
		rawHost           string
		hasPort, hasProto bool
		matches           = addrPattern.FindStringSubmatch(addr)
		subexpNames       = addrPattern.SubexpNames()
		groups            = make(map[string]string)
	)

	for i, value := range matches {
		groups[subexpNames[i]] = value
	}

	rawHost = groups["host"]
	hasPort = strings.ContainsRune(rawHost, ':')

	if hasPort {
		host, port, err := net.SplitHostPort(rawHost)
		if err != nil {
			return nil, err
		}
		groups["host"] = host
		groups["port"] = port
	} else if proto, hasProto = groups["proto"]; hasProto {
		if port, knownProto := protoPort[strings.ToLower(proto)]; knownProto {
			groups["host"] = rawHost
			groups["port"] = port
		} else {
			if proto == "" {
				return nil, fmt.Errorf("neither port nor protocol is given")
			}
			return nil, fmt.Errorf("port not given and protocol is unknown: %q", proto)
		}
	}

	if rawFreq, hasFreq := groups["freq"]; hasFreq && rawFreq != "" {
		freq, err := time.ParseDuration(rawFreq)
		if err != nil {
			return nil, err
		}
		pollFreq = freq
	}

	return &TCPSpec{Host: groups["host"], Port: groups["port"], PollFreq: pollFreq}, nil
}

// AllTCP waits until connections can be made to all given TCP addresses.
func AllTCP(
	rawAddrs []string,
	waitTimeout, pollFreq, statusFreq time.Duration,
	isQuiet bool,
) (time.Duration, error) {

	// Parse addresses into TCP specs.
	specs := make([]*TCPSpec, len(rawAddrs))
	addrs := make([]string, len(rawAddrs))
	for i, rawAddr := range rawAddrs {
		spec, err := ParseTCPSpec(rawAddr, pollFreq)
		if err != nil {
			return 0, err
		}
		specs[i] = spec
		addrs[i] = net.JoinHostPort(spec.Host, spec.Port)
	}

	var (
		showStatus         func(TCPMessage)
		pendingSet         = newPendingSet(addrs)
		ready              = make(chan TCPMessage)
		waiting            = make(chan TCPMessage)
		startTime, timeout = time.Now(), time.NewTimer(waitTimeout)
	)
	defer timeout.Stop()

	if isQuiet {
		// Set status freq to be double the wait timeout, preventing any status from being emitted.
		statusFreq = 2 * waitTimeout
		// Set showStatus to do nothing; needed even without any status being emitted to prevent
		// runtime nil deref runtime error.
		showStatus = func(msg TCPMessage) {}
	} else {
		statusVerb := mkFmtVerb([]string{Ready.String(), Waiting.String()}, 0, false)
		addrVerb := mkFmtVerb(addrs, 2, true)
		msgFmt := fmt.Sprintf("%s: %s [%%s] ...\n", statusVerb, addrVerb)
		showStatus = func(msg TCPMessage) {
			fmt.Printf(msgFmt, msg.Status, msg.Addr, msg.SinceStart())
		}
	}

	for _, spec := range specs {
		go SingleTCP(net.JoinHostPort(spec.Host, spec.Port), waiting, ready, spec.PollFreq, statusFreq, startTime)
	}

	for {
		select {
		case <-timeout.C:
			return 0, fmt.Errorf("reached timeout limit of %s", waitTimeout)

		case waitMsg := <-waiting:
			showStatus(waitMsg)

		case readyMsg := <-ready:
			if readyMsg.Err != nil {
				return 0, readyMsg.Err
			}

			pendingSet.Remove(readyMsg.Addr)
			showStatus(readyMsg)

			if pendingSet.IsEmpty() {
				return readyMsg.SinceStart(), nil
			}
		}
	}
}
