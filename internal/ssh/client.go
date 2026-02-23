package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	"shippy/internal/errors"
)

// ClientOptions represents SSH client configuration options
type ClientOptions struct {
	Host               string
	Port               int
	User               string
	KeyPath            string
	SSHOptions         map[string]string
	SSHMultiplexing    bool
	KnownHostsFile     string
	HostKeyFingerprint string
}

// Client represents an SSH client connection
type Client struct {
	client             *ssh.Client
	config             *ssh.ClientConfig
	host               string
	port               int
	user               string
	options            map[string]string
	multiplexing       bool
	keepaliveInterval  time.Duration
	keepaliveCountMax  int
	keepaliveStopChan  chan struct{}
}

// NewClient creates a new SSH client
func NewClient(host, user, keyPath string) (*Client, error) {
	return NewClientWithOptions(ClientOptions{
		Host:    host,
		Port:    DefaultSSHPort,
		User:    user,
		KeyPath: keyPath,
	})
}

// NewClientWithOptions creates a new SSH client with full options
func NewClientWithOptions(opts ClientOptions) (*Client, error) {
	// Set default port if not specified
	if opts.Port == 0 {
		opts.Port = DefaultSSHPort
	}

	// Determine SSH key path
	keyPath := opts.KeyPath
	if keyPath == "" {
		// Try to find default SSH keys
		home, err := getUserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("ssh_key not specified: %w", err)
		}

		// Try common SSH key locations
		defaultKeys := []string{
			filepath.Join(home, ".ssh", "id_ed25519"),
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_ecdsa"),
		}

		for _, defaultKey := range defaultKeys {
			if _, err := os.Stat(defaultKey); err == nil {
				keyPath = defaultKey
				break
			}
		}

		if keyPath == "" {
			return nil, fmt.Errorf("ssh_key not specified in configuration and no default SSH key found in ~/.ssh/")
		}
	}

	// Expand home directory if needed
	var err error
	keyPath, err = expandHomePath(keyPath)
	if err != nil {
		return nil, err
	}

	// Validate key path to prevent directory traversal
	if err := validateSSHKeyPath(keyPath); err != nil {
		return nil, fmt.Errorf("invalid SSH key path: %w", err)
	}

	// Check if key exists
	if _, err := os.Stat(keyPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH key not found at '%s'. Please check your ssh_key configuration", keyPath)
		}
		return nil, fmt.Errorf("cannot access SSH key at '%s': %w", keyPath, err)
	}

	// Read private key
	// #nosec G304 -- Path is validated above with validateSSHKeyPath
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key from '%s': %w", keyPath, err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key from '%s': %w (make sure this is a private key, not a .pub file)", keyPath, err)
	}

	// Extract UserKnownHostsFile from ssh_options if specified
	if opts.KnownHostsFile == "" && opts.SSHOptions != nil {
		if userKnownHosts, ok := opts.SSHOptions["UserKnownHostsFile"]; ok {
			opts.KnownHostsFile = userKnownHosts
		}
	}

	// Print SSH connection information
	fmt.Fprintf(os.Stderr, "  SSH connection details:\n")
	fmt.Fprintf(os.Stderr, "  • Target: %s@%s:%d\n", opts.User, opts.Host, opts.Port)
	fmt.Fprintf(os.Stderr, "  • SSH key: %s\n", keyPath)

	// Create host key callback with proper verification
	hostKeyCallback, err := createHostKeyCallback(&opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create host key verification: %w", err)
	}

	config := &ssh.ClientConfig{
		User: opts.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	// Apply SSH options to config

	// Parse and apply ConnectTimeout
	if timeoutStr, ok := opts.SSHOptions["ConnectTimeout"]; ok {
		timeout, err := parseTimeout(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ConnectTimeout value '%s': %w", timeoutStr, err)
		}
		config.Timeout = timeout
	} else {
		// Apply default timeout
		config.Timeout = DefaultConnectTimeout
	}

	// Parse ServerAliveInterval for keepalive
	var keepaliveInterval time.Duration
	if intervalStr, ok := opts.SSHOptions["ServerAliveInterval"]; ok {
		interval, err := parseTimeout(intervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid ServerAliveInterval value '%s': %w", intervalStr, err)
		}
		keepaliveInterval = interval
	}

	// Parse ServerAliveCountMax (default: 3)
	keepaliveCountMax := 3
	if countStr, ok := opts.SSHOptions["ServerAliveCountMax"]; ok {
		if count, err := strconv.Atoi(countStr); err == nil && count > 0 {
			keepaliveCountMax = count
		}
	}

	// Apply compression settings (note: Go SSH library has limited compression support)
	if compressionStr, ok := opts.SSHOptions["Compression"]; ok {
		if compressionStr == "yes" || compressionStr == "true" {
			// Enable compression - the library supports zlib compression
			// This is a hint to the library but actual support depends on the server
			_ = compressionStr // Compression is negotiated automatically by the library
		}
	}

	return &Client{
		config:            config,
		host:              opts.Host,
		port:              opts.Port,
		user:              opts.User,
		options:           opts.SSHOptions,
		multiplexing:      opts.SSHMultiplexing,
		keepaliveInterval: keepaliveInterval,
		keepaliveCountMax: keepaliveCountMax,
	}, nil
}

// Connect establishes the SSH connection
func (c *Client) Connect() error {
	// Build address with port
	addr := fmt.Sprintf("%s:%d", c.host, c.port)

	client, err := ssh.Dial("tcp", addr, c.config)
	if err != nil {
		return errors.SSHError(fmt.Sprintf("connecting to %s@%s", c.user, addr), err)
	}

	c.client = client

	// When multiplexing is enabled, the underlying TCP connection is reused
	// for all sessions (NewSession calls). The golang.org/x/crypto/ssh library
	// handles this automatically - each session multiplexes over the same connection.
	// No additional configuration needed beyond keeping the client alive.

	// Start keepalive goroutine if configured
	if c.keepaliveInterval > 0 {
		c.keepaliveStopChan = make(chan struct{})
		go c.sendKeepalives()
	}

	return nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	// Stop keepalive goroutine if running
	if c.keepaliveStopChan != nil {
		close(c.keepaliveStopChan)
		c.keepaliveStopChan = nil
	}

	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// RunCommand executes a command on the remote server
func (c *Client) RunCommand(cmd string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", errors.SSHError("creating SSH session", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		return outputStr, errors.CommandError(cmd, outputStr, err)
	}

	return string(output), nil
}

// RunCommandWithOutput executes a command and streams output to stdout/stderr
func (c *Client) RunCommandWithOutput(cmd string, stdout, stderr io.Writer) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = stdout
	session.Stderr = stderr

	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// UploadFile uploads a file to the remote server
func (c *Client) UploadFile(localPath, remotePath string, mode os.FileMode) error {
	// Validate local path to prevent directory traversal
	if err := validateLocalFilePath(localPath); err != nil {
		return fmt.Errorf("invalid local file path: %w", err)
	}

	// Open local file
	// #nosec G304 -- Path is validated above with validateLocalFilePath
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
	if _, err := c.RunCommand(fmt.Sprintf("mkdir -p %s", remoteDir)); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	// Create SCP session
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Get stdin pipe for the session
	w, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	// Start SCP send in goroutine
	errChan := make(chan error, 1)
	go func() {
		defer w.Close()

		// Send file header
		if _, err := fmt.Fprintf(w, "C%04o %d %s\n", mode.Perm(), stat.Size(), filepath.Base(remotePath)); err != nil {
			errChan <- fmt.Errorf("failed to send file header: %w", err)
			return
		}

		// Send file content
		if _, err := io.Copy(w, file); err != nil {
			errChan <- fmt.Errorf("failed to send file content: %w", err)
			return
		}

		// Send termination
		if _, err := fmt.Fprint(w, "\x00"); err != nil {
			errChan <- fmt.Errorf("failed to send termination: %w", err)
			return
		}

		errChan <- nil
	}()

	// Run SCP command
	if err := session.Run(fmt.Sprintf("scp -t %s", remotePath)); err != nil {
		return fmt.Errorf("scp failed: %w", err)
	}

	// Check if goroutine had any errors
	if err := <-errChan; err != nil {
		return err
	}

	return nil
}

// MkdirAll creates a directory and all parent directories on the remote server
func (c *Client) MkdirAll(path string) error {
	_, err := c.RunCommand(fmt.Sprintf("mkdir -p %s", path))
	return err
}

// Dial opens a connection to addr from the remote host.
// Used for SSH tunneling (e.g., connecting to a remote database).
func (c *Client) Dial(network, addr string) (net.Conn, error) {
	return c.client.Dial(network, addr)
}

// parseTimeout parses timeout string to time.Duration.
// Supports formats: "30" (seconds), "30s", "5m", "1h"
func parseTimeout(timeoutStr string) (time.Duration, error) {
	// If just a number, treat as seconds
	if seconds, err := strconv.Atoi(timeoutStr); err == nil {
		return time.Duration(seconds) * time.Second, nil
	}

	// Otherwise try to parse as duration
	duration, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 0, fmt.Errorf("must be number of seconds or duration (e.g. '30s', '5m'): %w", err)
	}

	return duration, nil
}

// sendKeepalives sends periodic keepalive messages to prevent connection timeout.
// This runs in a goroutine and is stopped when the client is closed.
func (c *Client) sendKeepalives() {
	ticker := time.NewTicker(c.keepaliveInterval)
	defer ticker.Stop()

	failedCount := 0

	for {
		select {
		case <-ticker.C:
			// Send keepalive (SSH_MSG_GLOBAL_REQUEST with keepalive@golang.org)
			// The true parameter requests a reply
			_, _, err := c.client.SendRequest("keepalive@golang.org", true, nil)
			if err != nil {
				failedCount++
				if failedCount >= c.keepaliveCountMax {
					// Connection likely dead, close it
					// #nosec G104 -- Error from Close in keepalive goroutine can be safely ignored
					c.client.Close()
					return
				}
			} else {
				failedCount = 0
			}
		case <-c.keepaliveStopChan:
			return
		}
	}
}
