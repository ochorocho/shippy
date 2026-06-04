package ssh

import (
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"
)

// proxyConn wraps a ProxyCommand subprocess as a net.Conn.
// The subprocess establishes the TCP transport; the SSH handshake runs over
// its stdio. This lets a trusted system binary (e.g. /usr/bin/nc) own the
// TCP connection when a network security agent blocks direct Go connections.
type proxyConn struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	remote string
}

func (c *proxyConn) Read(b []byte) (int, error)  { return c.stdout.Read(b) }
func (c *proxyConn) Write(b []byte) (int, error) { return c.stdin.Write(b) }

func (c *proxyConn) Close() error {
	err1 := c.stdin.Close()
	err2 := c.stdout.Close()
	_ = c.cmd.Wait() // exit status of the proxy process is not meaningful on close
	if err1 != nil {
		return err1
	}
	return err2
}

func (c *proxyConn) LocalAddr() net.Addr                      { return &pipeAddr{} }
func (c *proxyConn) RemoteAddr() net.Addr                     { return &pipeAddr{c.remote} }
func (c *proxyConn) SetDeadline(_ time.Time) error            { return nil }
func (c *proxyConn) SetReadDeadline(_ time.Time) error        { return nil }
func (c *proxyConn) SetWriteDeadline(_ time.Time) error       { return nil }

type pipeAddr struct{ s string }

func (a pipeAddr) Network() string { return "pipe" }
func (a pipeAddr) String() string  { return a.s }

// expandProxyTokens replaces OpenSSH-style tokens in a ProxyCommand string.
func expandProxyTokens(command, host string, port int) string {
	r := strings.NewReplacer(
		"%%", "%",
		"%h", host,
		"%p", fmt.Sprintf("%d", port),
	)
	return r.Replace(command)
}

// dialProxy runs command via the shell and returns a net.Conn backed by its stdio.
// command may contain %h/%p tokens (expanded to host/port) and shell syntax.
func dialProxy(command, host string, port int) (net.Conn, error) {
	expanded := expandProxyTokens(command, host, port)

	// Run through sh so quoted args and shell constructs work, matching OpenSSH behaviour.
	// #nosec G204 -- ProxyCommand is from user-controlled config (~/.ssh/config or .shippy.yaml)
	cmd := exec.Command("sh", "-c", expanded)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("ProxyCommand stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("ProxyCommand stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("ProxyCommand start %q: %w", expanded, err)
	}

	return &proxyConn{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		remote: fmt.Sprintf("%s:%d", host, port),
	}, nil
}
