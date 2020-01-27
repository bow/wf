package wait

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

var (
	// addrPattern is used for parsing input TCP addresses and extracting the relevant parts.
	addrPattern = regexp.MustCompile(
		"^(?P<schema>(?P<proto>[A-Za-z]+)://)?(?P<host>[^#]+)(#(?P<freq>.+))?",
	)
	// protoPort is a mapping between popular TCP-backed protocol names to their default port
	// numbers.
	protoPort = map[string]string{
		"amqp":       "5672",
		"amqps":      "5671",
		"http":       "80",
		"https":      "443",
		"imap":       "143",
		"mysql":      "3306",
		"ldap":       "389",
		"ldaps":      "636",
		"postgresql": "5432",
		"smtp":       "25",
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

// Addr returns the host and port of the TCP specifications, joined by ':'.
func (spec *TCPSpec) Addr() string {
	return net.JoinHostPort(spec.Host, spec.Port)
}

// Message is the interface for messages sent by the wait operations.
type Message interface {
	// Status returns the status of the message.
	Status() Status
	// Target returns the entity being waited.
	Target() string
	// Err returns an error, if the message contains any.
	Err() error
	// ElapsedTime returns the duration of the wait operation at the time of message creation.
	ElapsedTime() time.Duration
}

// TCPMessage is a container for wait operations on TCP servers.
type TCPMessage struct {
	// spec is the wait operation specifications.
	spec *TCPSpec
	// status is the wait operation status.
	status Status
	// startTime is when the wait operation starts.
	startTime time.Time
	// emitTime is when the message is created and emitted. The current implementation creates and
	// emits at the same time.
	emitTime time.Time
	// err is any operation that may have occurred.
	err error
}

// newTCPMessageStart creates a new TCPMessage with status Start and no errors.
func newTCPMessageStart(spec *TCPSpec, startTime time.Time) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Start,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       nil,
	}
}

// newTCPMessageReady creates a new TCPMessage with status Ready and no errors.
func newTCPMessageReady(spec *TCPSpec, startTime time.Time) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Ready,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       nil,
	}
}

// newTCPMessage failed creates a new TCPMessage with status Failed and the given error.
func newTCPMessageFailed(spec *TCPSpec, startTime time.Time, err error) *TCPMessage {
	return &TCPMessage{
		spec:      spec,
		status:    Failed,
		startTime: startTime,
		emitTime:  time.Now(),
		err:       err,
	}
}

// Status returns the status of the message.
func (msg *TCPMessage) Status() Status {
	return msg.status
}

// Target returns the target of the wait operation, which is `tcp://` prepended to Addr. If the
// specifications is nil, this returns `<none>`.
func (msg *TCPMessage) Target() string {
	if msg.spec == nil {
		return "<none>"
	}
	return "tcp://" + msg.Addr()
}

// Addr returns the address being waited. If the specifications is nil, this returns `<none>`.
func (msg *TCPMessage) Addr() string {
	if msg.spec == nil {
		return "<none>"
	}
	return msg.spec.Addr()
}

// ElapsedTime is the duration between waiting operation start and status emission.
func (msg *TCPMessage) ElapsedTime() time.Duration {
	return msg.emitTime.Sub(msg.startTime)
}

// Err returns the error contained in the message, if any.
func (msg *TCPMessage) Err() error {
	return msg.err
}

// ctxKey is the key type for wait contexts.
type ctxKey int

// startTimeCtxKey is the key for retrieving wait operation start time from contexts.
const startTimeCtxKey ctxKey = 0

// newContext creates a new context containing current time along with a cancellation function,
// based on the background context.
func newContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return context.WithValue(ctx, startTimeCtxKey, time.Now()), cancel
}

// startTimeFromContext extracts the wait operation start time from the given context. If the
// expected value does not exist or it does not typecheck, the current time is returned.
func startTimeFromContext(ctx context.Context) time.Time {
	startTime, ok := ctx.Value(startTimeCtxKey).(time.Time)
	if !ok {
		return time.Now()
	}
	return startTime
}

// ParseTCPSpec parses the given address into a TCPSpec and then returns a pointer to it. The
// address can be given in several forms: `<host>:<port>`, `<protocol>://<host>`, or
// `<protocol>://<host>:<port>`. For the second form, if the protocol is known, the port will be
// inferred from it (e.g. port 80 for HTTP and 443 for HTTPS). For the last form, the `<protocol>`
// is ignored.  This function also takes a `defaultPollFreq` argument, which it will use as the poll
// frequency of the TCPSpec if the raw address does not specify a poll frequency value.  The poll
// frequency value in the raw address is the string value of time.Duration, appended to the address
// after a `#` sign.
func ParseTCPSpec(rawAddr string, defaultPollFreq time.Duration) (*TCPSpec, error) {
	var (
		proto             string
		rawHost           string
		hasPort, hasProto bool
		matches           = addrPattern.FindStringSubmatch(rawAddr)
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
		defaultPollFreq = freq
	}

	return &TCPSpec{
		Host:     groups["host"],
		Port:     groups["port"],
		PollFreq: defaultPollFreq,
	}, nil
}

// ParseTCPSpecs parses multiple addresses into separate TCPSpecs, returned as a slice of pointers.
// It has the same semantics as `ParseTCPSpec`, only it works with multiple addresses instead of
// one.
func ParseTCPSpecs(rawAddrs []string, defaultPollFreq time.Duration) ([]*TCPSpec, error) {
	specs := make([]*TCPSpec, len(rawAddrs))

	for i, rawAddr := range rawAddrs {
		spec, err := ParseTCPSpec(rawAddr, defaultPollFreq)
		if err != nil {
			return []*TCPSpec{}, err
		}
		specs[i] = spec
	}

	return specs, nil
}

// SingleTCP waits until a TCP connection can be made to an address, attempting a connection every
// defined interval. Both of these are contained in the given specifications. It also accepts a
// context function, which it uses to listen to cancellation events from the parent context.
// The returned channel is closed after the wait operation has finished or if the parent context is
// cancelled.
func SingleTCP(ctx context.Context, spec *TCPSpec) <-chan *TCPMessage {
	startTime := startTimeFromContext(ctx)
	out := make(chan *TCPMessage, 2)

	checkConn := func() *TCPMessage {
		_, err := net.DialTimeout("tcp", spec.Addr(), spec.PollFreq)

		if err == nil {
			return newTCPMessageReady(spec, startTime)
		}
		if shouldWait(err) {
			return nil
		}
		return newTCPMessageFailed(spec, startTime, err)
	}

	go func() {
		pollTicker := time.NewTicker(spec.PollFreq)
		defer pollTicker.Stop()

		defer close(out)

		out <- newTCPMessageStart(spec, startTime)

		// So that we start polling immediately, without waiting for the first tick.
		// There is no way to do this via the current ticker API.
		// See: https://github.com/golang/go/issues/17601
		if msg := checkConn(); msg != nil {
			out <- msg
			return
		}

		for {
			select {
			case <-ctx.Done():
				out <- newTCPMessageFailed(spec, startTime, ctx.Err())
				return

			case <-pollTicker.C:
				if msg := checkConn(); msg != nil {
					out <- msg
					return
				}
			}
		}
	}()

	return out
}

// AllTCP waits until connections can be made to all given TCP input specifications for at most
// `waitTimeout` long. It returns a channel through which all wait operation-related messages will
// be sent.  The returned channel is closed after all wait operations have finished.
func AllTCP(specs []*TCPSpec, waitTimeout time.Duration) <-chan *TCPMessage {

	addrs := make([]string, len(specs))
	for i, spec := range specs {
		addrs[i] = spec.Addr()
	}

	var (
		chs         = make([](<-chan *TCPMessage), len(specs))
		out         = make(chan *TCPMessage)
		ctx, cancel = newContext()
	)

	for i, spec := range specs {
		chs[i] = SingleTCP(ctx, spec)
	}

	msgs := merge(chs)
	timeout := time.NewTimer(waitTimeout)

	go func() {
		defer timeout.Stop()
		defer cancel()
		defer close(out)

		for {
			select {
			case <-timeout.C:
				msg := newTCPMessageFailed(
					nil,
					startTimeFromContext(ctx),
					fmt.Errorf("exceeded timeout limit of %s", waitTimeout),
				)
				out <- msg
				return

			case msg, isOpen := <-msgs:
				if !isOpen {
					return
				}
				out <- msg
			}
		}
	}()

	return out
}
