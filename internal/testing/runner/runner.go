// Package runner runs the compiled language server for an acceptance test.
//
// It is the small equivalent of backplane-go's testing/runner: this server is
// not an HTTP service, so there is no readiness endpoint to poll and no
// liveness port to discover. Over stdio there is nothing to discover at all;
// over a socket the port is read from the line the server prints for its
// client.
//
// Both transports are here because both ship. Stdio is what an editor uses and
// what the bulk of a test suite should run over; the socket path exists so
// that the startup line the extension waits for is covered by something.
package runner

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"
)

// startTimeout bounds how long a server may take to announce its port. The
// server sleeps a second before printing the line, so this is generous.
const startTimeout = 30 * time.Second

// startedLine is what the server prints once it is listening. The extension
// waits for it before connecting, so its shape is a contract.
var startedLine = regexp.MustCompile(`^Server started on port (\d+)`)

// Server is a running language server.
type Server struct {
	cmd    *exec.Cmd
	stream io.ReadWriteCloser
	stderr *syncBuffer
	stdout *syncBuffer
	port   int

	stopOnce sync.Once
}

// StartStdio runs the binary with --stdio and talks to it over its pipes,
// which is how an editor runs it. The server is stopped when the test ends.
func StartStdio(t *testing.T, binary string, environment ...string) *Server {
	t.Helper()

	server := command(t, binary, []string{"--stdio"}, environment)

	stdin, err := server.cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}

	stdout, err := server.cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	server.stream = &pipes{reader: stdout, writer: stdin}

	if err := server.cmd.Start(); err != nil {
		t.Fatalf("starting %s: %v", binary, err)
	}

	register(t, server)

	return server
}

// StartSocket runs the binary as a socket server on a port it chooses, waits
// for the line it prints once it is listening, and dials it. The server is
// stopped when the test ends.
func StartSocket(t *testing.T, binary string, environment ...string) *Server {
	t.Helper()

	server := command(t, binary, []string{"--host", "127.0.0.1", "--port", "0"}, environment)

	stdout, err := server.cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}

	if err := server.cmd.Start(); err != nil {
		t.Fatalf("starting %s: %v", binary, err)
	}

	register(t, server)

	ports := make(chan int, 1)
	go server.scanForPort(stdout, ports)

	select {
	case port := <-ports:
		server.port = port
	case <-time.After(startTimeout):
		t.Fatalf("server did not report a port within %s; stderr:\n%s", startTimeout, server.Stderr())
	}

	connection, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", server.port), startTimeout)
	if err != nil {
		t.Fatalf("dialing the server: %v", err)
	}
	server.stream = connection

	return server
}

// Stream is the server's connection, ready for a jsonrpc2 client.
func (s *Server) Stream() io.ReadWriteCloser {
	return s.stream
}

// Port is the port a socket server is listening on, and zero over stdio.
func (s *Server) Port() int {
	return s.port
}

// Stdout is everything the server has printed to stdout. Over stdio that is
// the protocol itself, so it is only meaningful for a socket server.
func (s *Server) Stdout() string {
	return s.stdout.String()
}

// Stderr is everything the server has logged. The server logs at debug level
// by default, so this includes a line per method it handled.
func (s *Server) Stderr() string {
	return s.stderr.String()
}

// Stop closes the connection and ends the process. It runs when the test ends
// and is safe to call again.
func (s *Server) Stop() {
	s.stopOnce.Do(func() {
		if s.stream != nil {
			_ = s.stream.Close()
		}
		if s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		_ = s.cmd.Wait()
	})
}

// command builds the process without starting it, with its own home and
// temporary directory: the server writes fetched orb sources under the
// temporary directory, and tests must not leave them in the developer's.
func command(t *testing.T, binary string, args []string, environment []string) *Server {
	t.Helper()

	home := t.TempDir()

	cmd := exec.Command(binary, args...)
	cmd.Env = append(cmd.Environ(),
		"HOME="+home,
		"TMPDIR="+home,
	)
	cmd.Env = append(cmd.Env, environment...)

	server := &Server{
		cmd:    cmd,
		stderr: &syncBuffer{},
		stdout: &syncBuffer{},
	}
	cmd.Stderr = server.stderr

	return server
}

// register stops the server when the test ends, and reports what it logged if
// the test failed, which is where a server-side panic shows up.
func register(t *testing.T, server *Server) {
	t.Helper()

	t.Cleanup(func() {
		server.Stop()

		if t.Failed() {
			t.Logf("server stderr:\n%s", server.Stderr())
		}
	})
}

// scanForPort reads stdout until the server announces its port, keeping
// everything it read for Stdout.
func (s *Server) scanForPort(stdout io.Reader, ports chan<- int) {
	scanner := bufio.NewScanner(stdout)
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		_, _ = s.stdout.Write([]byte(line + "\n"))

		if found {
			continue
		}

		if match := startedLine.FindStringSubmatch(line); match != nil {
			port, err := strconv.Atoi(match[1])
			if err != nil {
				continue
			}
			found = true
			ports <- port
		}
	}
}

// pipes is the process's stdout and stdin as one stream.
type pipes struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (p *pipes) Read(b []byte) (int, error) {
	return p.reader.Read(b)
}

func (p *pipes) Write(b []byte) (int, error) {
	return p.writer.Write(b)
}

// Close closes the writer first, which is what tells the server its client has
// gone, and then the reader.
func (p *pipes) Close() error {
	writeErr := p.writer.Close()
	readErr := p.reader.Close()

	if writeErr != nil {
		return writeErr
	}

	return readErr
}

// syncBuffer collects output written by the process's goroutines while a test
// reads it.
type syncBuffer struct {
	mu      sync.Mutex
	content []byte
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.content = append(b.content, p...)

	return len(p), nil
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	return string(b.content)
}
