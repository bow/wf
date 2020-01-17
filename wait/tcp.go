package wait

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
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

func (spec *TCPSpec) Addr() string {
	return net.JoinHostPort(spec.Host, spec.Port)
}

type TCPMessage struct {
	spec      *TCPSpec
	status    Status
	startTime time.Time
	emitTime  time.Time
	err       error
}

func NewTCPMessageReady(spec *TCPSpec, startTime time.Time) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Ready,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       nil,
	}
}

func NewTCPMessageWaiting(spec *TCPSpec, startTime time.Time) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Waiting,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       nil,
	}
}

func NewTCPMessageFailed(spec *TCPSpec, startTime time.Time, err error) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Failed,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       err,
	}
}

func (msg *TCPMessage) Status() Status {
	return msg.status
}

// Addr is the address being waited.
func (msg *TCPMessage) Addr() string {
	return msg.spec.Addr()
}

// ElapsedTime is the duration between waiting operation start and status emission.
func (msg *TCPMessage) ElapsedTime() time.Duration {
	return msg.emitTime.Sub(msg.startTime)
}

func (msg *TCPMessage) Err() error {
	return msg.err
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

// SingleTCP waits until a TCP connection can be made to the given address.
func SingleTCP(spec *TCPSpec, statusFreq time.Duration, startTime time.Time) <-chan *TCPMessage {
	if startTime.IsZero() {
		startTime = time.Now()
	}

	out := make(chan *TCPMessage, 1)

	checkConn := func() bool {
		_, err := net.DialTimeout("tcp", spec.Addr(), spec.PollFreq)

		if err == nil {
			out <- NewTCPMessageReady(spec, startTime)
			return true
		}

		if shouldWait(err) {
			return false
		}

		out <- NewTCPMessageFailed(spec, startTime, err)
		return true
	}

	go func() {
		pollTicker := time.NewTicker(spec.PollFreq)
		defer pollTicker.Stop()

		statusTicker := time.NewTicker(statusFreq)
		defer statusTicker.Stop()

		defer close(out)

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
				out <- NewTCPMessageWaiting(spec, startTime)
			}
		}
	}()

	return out
}

// AllTCP waits until connections can be made to all given TCP addresses.
func AllTCP(
	rawAddrs []string,
	waitTimeout, pollFreq, statusFreq time.Duration,
	isQuiet bool,
) (time.Duration, error) {

	// Parse addresses into TCP specs.
	addrs := make([]string, len(rawAddrs))
	specs := make([]*TCPSpec, len(rawAddrs))
	for i, rawAddr := range rawAddrs {
		spec, err := ParseTCPSpec(rawAddr, pollFreq)
		if err != nil {
			return 0, err
		}
		specs[i] = spec
		addrs[i] = spec.Addr()
	}

	var (
		showStatus         func(*TCPMessage)
		chs                = make([](<-chan *TCPMessage), len(specs))
		startTime, timeout = time.Now(), time.NewTimer(waitTimeout)
	)
	defer timeout.Stop()

	if isQuiet {
		// Set status freq to be double the wait timeout, preventing any status from being emitted.
		statusFreq = 2 * waitTimeout
		// Set showStatus to do nothing; needed even without any status being emitted to prevent
		// runtime nil deref runtime error.
		showStatus = func(msg *TCPMessage) {}
	} else {
		statusVerb := mkFmtVerb(statusValues, 0, false)
		addrVerb := mkFmtVerb(addrs, 2, true)
		msgFmt := fmt.Sprintf("%s: %s [%%s] ...\n", statusVerb, addrVerb)
		showStatus = func(msg *TCPMessage) {
			fmt.Printf(msgFmt, msg.Status(), msg.Addr(), msg.ElapsedTime())
		}
	}

	for i, spec := range specs {
		chs[i] = SingleTCP(spec, statusFreq, startTime)
	}

	msgs := merge(chs)
	// lastMsg keeps track of the last emitted message so that when the merged channel is closed,
	// we can still emit the ElapsedTime() of the total wait operation (proxied here as the longest
	// waiting time, i.e. the ElapsedTime() of the last message).
	var lastMsg *TCPMessage

	for {
		select {
		case <-timeout.C:
			return 0, fmt.Errorf("reached timeout limit of %s", waitTimeout)

		case msg, isOpen := <-msgs:
			if !isOpen {
				return lastMsg.ElapsedTime(), nil
			}
			lastMsg = msg
			switch msg.Status() {
			case Waiting:
				showStatus(msg)
			case Failed:
				return 0, msg.Err()
			case Ready:
				showStatus(msg)
			}
		}
	}
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
