package wait

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestMessageTarget(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name string
		in   Message
		want string
	}{
		{
			"with TCPSpec",
			newTCPMessageReady(
				&TCPSpec{Host: "localhost", Port: "7000", PollFreq: 1 * time.Second},
				time.Now(),
			),
			"tcp://localhost:7000",
		},
		{
			"no TCPSpec",
			newTCPMessageFailed(nil, time.Now(), fmt.Errorf("stub")),
			"<none>",
		},
	}

	for i, test := range tests {
		i := i
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			name := test.name
			want := test.want
			got := test.in.Target()

			if want != got {
				t.Errorf("test[%d] %q failed - want: %q, got: %q", i, name, want, got)
			}
		})
	}
}

func TestParseTCPSpec(t *testing.T) {
	t.Parallel()

	var commonPollFreq = 1 * time.Second
	var tests = []struct {
		name     string
		in       string
		wantSpec *TCPSpec
		wantErr  error
	}{
		{
			"no protocol, no port",
			"localhost",
			nil,
			fmt.Errorf("neither port nor protocol is given"),
		},
		{
			"unknown protocol, no port",
			"foo://localhost",
			nil,
			fmt.Errorf("port not given and protocol is unknown: \"foo\""),
		},
		{
			"no protocol, port, no poll freq",
			"localhost:5000",
			&TCPSpec{
				Host:     "localhost",
				Port:     "5000",
				PollFreq: commonPollFreq,
			},
			nil,
		},
		{
			"no protocol, port, poll freq",
			"localhost:5000#3s",
			&TCPSpec{
				Host:     "localhost",
				Port:     "5000",
				PollFreq: 3 * time.Second,
			},
			nil,
		},
		{
			"http, no port, no poll freq",
			"http://localhost",
			&TCPSpec{
				Host:     "localhost",
				Port:     "80",
				PollFreq: commonPollFreq,
			},
			nil,
		},
		{
			"http, no port, poll freq",
			"http://localhost#500ms",
			&TCPSpec{
				Host:     "localhost",
				Port:     "80",
				PollFreq: 500 * time.Millisecond,
			},
			nil,
		},
		{
			"http, port, no poll freq",
			"http://localhost:3000",
			&TCPSpec{
				Host:     "localhost",
				Port:     "3000",
				PollFreq: commonPollFreq,
			},
			nil,
		},
		{
			"http, port, poll freq",
			"http://localhost:3000#2s",
			&TCPSpec{
				Host:     "localhost",
				Port:     "3000",
				PollFreq: 2 * time.Second,
			},
			nil,
		},
	}

	for i, test := range tests {
		i := i
		test := test

		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			name := test.name
			wantSpec := test.wantSpec
			wantErr := test.wantErr
			gotSpec, gotErr := ParseTCPSpec(test.in, commonPollFreq)

			if wantErr != nil && gotErr.Error() != wantErr.Error() {
				t.Errorf("test[%d] %q failed - want err: %q, got: %q", i, name, wantErr, gotErr)
			}

			if wantErr == nil && *wantSpec != *gotSpec {
				t.Errorf(
					"test[%d] %q failed - want spec: %+v, got: %+v",
					i,
					name,
					*wantSpec,
					*gotSpec,
				)
			}
		})
	}
}

func ExampleParseTCPSpec() {
	spec, _ := ParseTCPSpec("golang.org:80", 1*time.Second)
	fmt.Println("host:", spec.Host)
	fmt.Println("port:", spec.Port)
	fmt.Println("poll freq:", spec.PollFreq)
	// Output:
	// host: golang.org
	// port: 80
	// poll freq: 1s
}

func ExampleParseTCPSpec_proto() {
	spec, _ := ParseTCPSpec("https://golang.org", 1*time.Second)
	fmt.Println("host:", spec.Host)
	fmt.Println("port:", spec.Port)
	fmt.Println("poll freq:", spec.PollFreq)
	// Output:
	// host: golang.org
	// port: 443
	// poll freq: 1s
}

func ExampleParseTCPSpec_freq() {
	spec, _ := ParseTCPSpec("amqps://127.0.0.1#500ms", 1*time.Second)
	fmt.Println("host:", spec.Host)
	fmt.Println("port:", spec.Port)
	fmt.Println("poll freq:", spec.PollFreq)
	// Output:
	// host: 127.0.0.1
	// port: 5671
	// poll freq: 500ms
}

func TestParseTCPSpecs(t *testing.T) {
	t.Parallel()

	var commonPollFreq = 1 * time.Second
	var tests = []struct {
		name      string
		in        []string
		wantSpecs []*TCPSpec
		wantErr   error
	}{
		{
			"all ok",
			[]string{
				"127.0.0.1:3000",
				"https://golang.org",
				"localhost:1234#200ms",
			},
			[]*TCPSpec{
				{"127.0.0.1", "3000", 1 * time.Second},
				{"golang.org", "443", 1 * time.Second},
				{"localhost", "1234", 200 * time.Millisecond},
			},
			nil,
		},
		{
			"some err",
			[]string{
				"127.0.0.1:3000",
				"localhost",
				"localhost:1234#200ms",
			},
			[]*TCPSpec{},
			fmt.Errorf("address 1: neither port nor protocol is given"),
		},
	}

	for i, test := range tests {
		i := i
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			name := test.name
			wantSpecs := test.wantSpecs
			wantErr := test.wantErr

			gotSpecs, gotErr := ParseTCPSpecs(test.in, commonPollFreq)

			if wantErr != nil && gotErr.Error() != wantErr.Error() {
				t.Errorf("test[%d] %q failed - want error: %q, got: %q", i, name, wantErr, gotErr)
			}

			if len(wantSpecs) != len(gotSpecs) {
				t.Fatalf(
					"test[%d] %q failed - want: %d specs, got: %d",
					i,
					name,
					len(wantSpecs),
					len(gotSpecs),
				)
			}
			for j, wantSpec := range wantSpecs {
				gotSpec := gotSpecs[j]
				if wantErr == nil && *wantSpec != *gotSpec {
					t.Errorf(
						"test[%d][%d] %q failed - got spec: %+v, want: %+v",
						i,
						j,
						name,
						*gotSpec,
						*wantSpec,
					)
				}
			}
		})
	}
}

// tcpServerHost is the hostname for the test TCP server.
const tcpServerHost = "127.0.0.1"

// getLocalTCPPort returns a TCP port for testing by asking the kernel for a free port.
func getLocalTCPPort() string {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(tcpServerHost, "0"))
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	return strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
}

// tcpServer is a wrapper struct for launching test TCP servers.
type tcpServer struct {
	host, port string
	// readyDelay is the duration to wait before the server is running.
	readyDelay time.Duration
	t          *testing.T
}

// addr returns the tcpServer address.
func (srv *tcpServer) addr() string {
	return net.JoinHostPort(srv.host, srv.port)
}

// start starts the test TCP server. It returns a context.Context value based on the input context,
// along with a cancellation function for stopping the server and ensuring proper cleanup.
func (srv *tcpServer) start(ctx context.Context) (context.Context, context.CancelFunc) {
	ictx, icancel := context.WithCancel(ctx)

	go func(gctx context.Context, t *testing.T, addr string, delay time.Duration) {
		t.Helper()
		select {
		// Handle case when the goroutine needs to be killed prior to server start.
		case <-gctx.Done():
			return
		// Expected flow: wait for `delay` before starting the server.
		case <-time.After(delay):
		}

		listener, err := net.Listen("tcp", addr)
		if err != nil {
			t.Logf("failed starting test TCP server %q: %s", addr, err)
			return
		}
		defer listener.Close()

		for {
			conn, err := listener.Accept()
			if err != nil {
				t.Logf("failed accepting TCP connection %q: %s", addr, err)
				return
			}
			select {
			case <-gctx.Done():
				conn.Close()
				return
			default:
			}
		}
	}(ictx, srv.t, srv.addr(), srv.readyDelay)

	return ictx, func() {
		var addr = srv.addr()
		icancel()
		// Dial to the server so that listener.Accept progresses and the ctx.Done() case is
		// selected.
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return
		}
		conn.Close()
	}
}

// tcpServerGroup is a helper container for starting multiple TCP servers.
type tcpServerGroup struct {
	servers []*tcpServer
	t       *testing.T
}

// start starts all the TCP servers in the group, ensuring they do so at the same time. It returns a
// context.Context based on the input context, along with a cancellation function for stopping all
// the servers and ensuring proper cleanup.
func (grp *tcpServerGroup) start(ctx context.Context) (context.Context, context.CancelFunc) {
	var (
		wgStart, wgEnd sync.WaitGroup
		ictx, icancel  = context.WithCancel(ctx)
	)

	// Track start and end jobs.
	wgStart.Add(1)
	wgEnd.Add(1)

	for _, srv := range grp.servers {
		go func(srv *tcpServer, ictx context.Context, wgStart, wgEnd *sync.WaitGroup) {
			wgStart.Wait()
			_, cancel := srv.start(ictx)
			// Wait until outer scope calls wgEnd.Done.
			wgEnd.Wait()
			cancel()
		}(srv, ictx, &wgStart, &wgEnd)
	}
	// Start all servers at the same time.
	wgStart.Done()

	return ictx, func() {
		icancel()
		// Release wgEnd.Wait() block in all launched goroutines.
		wgEnd.Done()
	}
}

// messageBox is a test helper container for messages emitted by the wait operations.
type messageBox struct {
	msgs []Message
}

// newMessageBox creates a messageBox by draining all the messages from the given channel.
func newMessageBox(ch <-chan *TCPMessage) *messageBox {
	msgs := make([]Message, 0)
	for msg := range ch {
		msgs = append(msgs, msg)
	}
	return &messageBox{msgs: msgs}
}

// count returns the number of messages in the box.
func (mb *messageBox) count() int {
	return len(mb.msgs)
}

// filterByTCPAddr returns a new message box containing only TCPMessages with the given address.
func (mb *messageBox) filterByTCPAddr(addr string) *messageBox {
	filtered := make([]Message, 0)
	for _, msg := range mb.msgs {
		if tcpMsg, isTCPMessage := msg.(*TCPMessage); isTCPMessage && tcpMsg.Addr() == addr {
			filtered = append(filtered, tcpMsg)
		}
	}
	return &messageBox{msgs: filtered}
}

func TestOneTCPReady(t *testing.T) {
	t.Parallel()

	var (
		waitTimeout = 3 * time.Second
		server      = &tcpServer{
			host:       tcpServerHost,
			port:       getLocalTCPPort(),
			readyDelay: 1 * time.Second,
			t:          t,
		}
		spec = &TCPSpec{Host: server.host, Port: server.port, PollFreq: 500 * time.Millisecond}
	)

	_, cancel := server.start(context.Background())
	defer cancel()

	msgs := OneTCP(spec, waitTimeout)

	// There must be 2 messages in total.
	mb := newMessageBox(msgs)
	if msgCount := mb.count(); msgCount != 2 {
		t.Fatalf("test failed - want %d messages, got %d", 2, msgCount)
	}

	// The last message's ElapsedTime must be at least equal to waitTimeout.
	if elTime := mb.msgs[mb.count()-1].ElapsedTime(); elTime >= waitTimeout {
		t.Errorf("test failed - elapsed time %s exceeded timeout limit of %s", elTime, waitTimeout)
	}

	// The messages from waiting for the server must be as expected.
	if status := mb.msgs[0].Status(); status != Start {
		t.Errorf("test msgs[0].Status() failed - want: %s, got %s", Start, status)
	}
	if status := mb.msgs[1].Status(); status != Ready {
		t.Errorf("test msgs[1].Status() failed - want: %s, got %s", Ready, status)
	}
}

func TestAllTCPReady(t *testing.T) {
	t.Parallel()

	var (
		waitTimeout = 5 * time.Second
		servers     = []*tcpServer{
			{tcpServerHost, getLocalTCPPort(), 0 * time.Second, t},
			{tcpServerHost, getLocalTCPPort(), 3 * time.Second, t},
		}
		group = tcpServerGroup{servers: servers, t: t}
	)

	_, cancel := group.start(context.Background())
	defer cancel()

	msgs := AllTCP(
		[]*TCPSpec{
			{servers[0].host, servers[0].port, 500 * time.Millisecond},
			{servers[1].host, servers[1].port, 500 * time.Millisecond},
		},
		waitTimeout,
	)

	// There must be 4 messages in total.
	mb := newMessageBox(msgs)
	if msgCount := mb.count(); msgCount != 4 {
		t.Fatalf("test failed - want %d messages, got %d", 4, msgCount)
	}

	// The last message's ElapsedTime must be less than waitTimeout.
	if elTime := mb.msgs[mb.count()-1].ElapsedTime(); elTime >= waitTimeout {
		t.Errorf("test failed - elapsed time %s exceeded timeout limit of %s", elTime, waitTimeout)
	}

	// The messages from waiting for the first server must be as expected.
	addr1 := servers[0].addr()
	mb1 := mb.filterByTCPAddr(addr1)
	if msgCount := mb1.count(); msgCount != 2 {
		t.Fatalf("test[%s] failed - want %d messages, got %d", addr1, 2, msgCount)
	}
	if status := mb1.msgs[0].Status(); status != Start {
		t.Errorf("test[%s] msgs[0].Status() failed - want: %s, got %s", addr1, Start, status)
	}
	if status := mb1.msgs[1].Status(); status != Ready {
		t.Errorf("test[%s] msgs[1].Status() failed - want: %s, got %s", addr1, Ready, status)
	}

	// The messages from waiting for the second server must be as expected.
	addr2 := servers[1].addr()
	mb2 := mb.filterByTCPAddr(addr2)
	if msgCount := mb2.count(); msgCount != 2 {
		t.Fatalf("test[%s] failed - want %d messages, got %d", addr2, 2, msgCount)
	}
	if status := mb2.msgs[0].Status(); status != Start {
		t.Errorf("test[%s] msgs[0].Status() failed - want: %s, got %s", addr2, Start, status)
	}
	if status := mb2.msgs[1].Status(); status != Ready {
		t.Errorf("test[%s] msgs[1].Status() failed - want: %s, got %s", addr2, Ready, status)
	}
}

func TestAllTCPTimeout(t *testing.T) {
	t.Parallel()

	var (
		waitTimeout = 5 * time.Second
		servers     = []*tcpServer{
			{tcpServerHost, getLocalTCPPort(), 10 * time.Second, t},
			{tcpServerHost, getLocalTCPPort(), 1 * time.Second, t},
		}
		group = tcpServerGroup{servers: servers, t: t}
	)

	_, cancel := group.start(context.Background())
	defer cancel()

	msgs := AllTCP(
		[]*TCPSpec{
			{servers[0].host, servers[0].port, 500 * time.Millisecond},
			{servers[1].host, servers[1].port, 500 * time.Millisecond},
		},
		waitTimeout,
	)

	// There must be 4 messages in total.
	mb := newMessageBox(msgs)
	if msgCount := mb.count(); msgCount != 4 {
		t.Fatalf("test failed - want %d messages, got %d", 4, msgCount)
	}

	// The last message's ElapsedTime must be at least equal to waitTimeout.
	if elTime := mb.msgs[mb.count()-1].ElapsedTime(); elTime < waitTimeout {
		t.Errorf(
			"test failed - elapsed time %s is less than timeout limit of %s",
			elTime,
			waitTimeout,
		)
	}
	// The last one must be a timeout failure.
	if status := mb.msgs[mb.count()-1].Status(); status != Failed {
		t.Errorf("test failed msgs[-1].Status() failed - want: %s, got: %s", Failed, status)
	}

	// The messages from waiting for the first server must be as expected.
	addr1 := servers[0].addr()
	mb1 := mb.filterByTCPAddr(addr1)
	if msgCount := mb1.count(); msgCount != 1 {
		t.Fatalf("test[%s] failed - want: %d messages, got: %d", addr1, 1, msgCount)
	}
	if status := mb1.msgs[0].Status(); status != Start {
		t.Errorf("test[%s] msgs[0].Status() failed - want: %s, got: %s", addr1, Start, status)
	}

	// The messages from waiting for the second server must be as expected.
	addr2 := servers[1].addr()
	mb2 := mb.filterByTCPAddr(addr2)
	if msgCount := mb2.count(); msgCount != 2 {
		t.Fatalf("test[%s] failed - want: %d messages, got: %d", addr2, 2, msgCount)
	}
	if status := mb2.msgs[0].Status(); status != Start {
		t.Errorf("test[%s] msgs[0].Status() failed - want: %s, got %s", addr2, Start, status)
	}
	if status := mb2.msgs[1].Status(); status != Ready {
		t.Errorf("test[%s] msgs[1].Status() failed - want: %s, got %s", addr2, Ready, status)
	}
}
