package ssh

import (
	"strings"
	"testing"
	"time"
)

func TestExpandProxyTokens(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		host     string
		port     int
		expected string
	}{
		{
			name:     "nc with host and port",
			command:  "/usr/bin/nc %h %p",
			host:     "sparkles.lan",
			port:     22,
			expected: "/usr/bin/nc sparkles.lan 22",
		},
		{
			name:     "nc with IP and custom port",
			command:  "/usr/bin/nc %h %p",
			host:     "192.168.1.218",
			port:     2222,
			expected: "/usr/bin/nc 192.168.1.218 2222",
		},
		{
			name:     "ssh ProxyJump style",
			command:  "ssh -W %h:%p jump.host",
			host:     "internal.host",
			port:     22,
			expected: "ssh -W internal.host:22 jump.host",
		},
		{
			name:     "percent escape",
			command:  "echo %%done",
			host:     "host",
			port:     22,
			expected: "echo %done",
		},
		{
			name:     "repeated tokens",
			command:  "nc %h %p # connecting to %h",
			host:     "myhost",
			port:     22,
			expected: "nc myhost 22 # connecting to myhost",
		},
		{
			name:     "no tokens",
			command:  "/usr/bin/nc myhost 22",
			host:     "ignored",
			port:     0,
			expected: "/usr/bin/nc myhost 22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandProxyTokens(tt.command, tt.host, tt.port)
			if got != tt.expected {
				t.Errorf("expandProxyTokens(%q, %q, %d) = %q, want %q",
					tt.command, tt.host, tt.port, got, tt.expected)
			}
		})
	}
}

func TestDialProxyReadsOutput(t *testing.T) {
	// echo outputs a fixed string immediately, no stdin needed.
	conn, err := dialProxy("echo SSH-2.0-test", "testhost", 22)
	if err != nil {
		t.Fatalf("dialProxy: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := strings.TrimSpace(string(buf[:n])); got != "SSH-2.0-test" {
		t.Errorf("Read = %q, want %q", got, "SSH-2.0-test")
	}
}

func TestDialProxyWriteRead(t *testing.T) {
	// cat echoes stdin back to stdout; closing stdin flushes the output.
	conn, err := dialProxy("cat", "testhost", 22)
	if err != nil {
		t.Fatalf("dialProxy: %v", err)
	}

	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	// Close stdin so cat flushes and exits.
	conn.(*proxyConn).stdin.Close()

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _ := conn.Read(buf)
	if got := string(buf[:n]); got != "hello" {
		t.Errorf("Read = %q, want %q", got, "hello")
	}
	conn.Close()
}

func TestDialProxyAddrs(t *testing.T) {
	conn, err := dialProxy("cat", "myhost", 2222)
	if err != nil {
		t.Fatalf("dialProxy: %v", err)
	}
	defer conn.Close()

	if got := conn.RemoteAddr().String(); got != "myhost:2222" {
		t.Errorf("RemoteAddr = %q, want %q", got, "myhost:2222")
	}
	if got := conn.RemoteAddr().Network(); got != "pipe" {
		t.Errorf("RemoteAddr.Network = %q, want %q", got, "pipe")
	}
}

func TestDialProxyTokensExpanded(t *testing.T) {
	// Verify tokens are expanded before the command runs.
	// printf %s echoes its argument without a trailing newline.
	conn, err := dialProxy("printf '%s' %h:%p", "myhost", 22)
	if err != nil {
		t.Fatalf("dialProxy: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 64)
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, _ := conn.Read(buf)
	if got := string(buf[:n]); got != "myhost:22" {
		t.Errorf("command output = %q, want %q", got, "myhost:22")
	}
}
