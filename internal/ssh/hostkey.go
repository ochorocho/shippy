package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// createHostKeyCallback creates an appropriate HostKeyCallback based on the provided options
func createHostKeyCallback(opts *ClientOptions) (ssh.HostKeyCallback, error) {
	// Check if user explicitly wants insecure mode
	strictHostKeyChecking := opts.SSHOptions["StrictHostKeyChecking"]
	if strictHostKeyChecking == "no" {
		fmt.Fprintf(os.Stderr, "  • Host key checking: disabled (insecure)\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: SSH host key verification disabled (StrictHostKeyChecking=no)\n")
		fmt.Fprintf(os.Stderr, "   This makes you vulnerable to man-in-the-middle attacks!\n")
		// #nosec G106 -- Intentionally insecure mode when user explicitly opts in with clear warning
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// If a specific host key fingerprint is provided, use it
	if opts.HostKeyFingerprint != "" {
		fmt.Fprintf(os.Stderr, "  • Host key checking: pinned fingerprint\n")
		fmt.Fprintf(os.Stderr, "  • Expected fingerprint: SHA256:%s\n", opts.HostKeyFingerprint)
		return createFingerprintCallback(opts.HostKeyFingerprint, opts.Host, opts.Port)
	}

	// Determine if we should auto-accept new host keys
	acceptNew := strictHostKeyChecking == "accept-new"

	// Use known_hosts file for verification
	knownHostsFile := opts.KnownHostsFile
	if knownHostsFile == "" {
		// Default to ~/.ssh/known_hosts
		var err error
		knownHostsFile, err = getDefaultKnownHostsFile()
		if err != nil {
			return nil, err
		}
	}

	// Show host key verification info
	if acceptNew {
		fmt.Fprintf(os.Stderr, "  • Host key checking: accept-new (auto-accept unknown hosts)\n")
	} else {
		fmt.Fprintf(os.Stderr, "  • Host key checking: enabled (secure)\n")
	}
	fmt.Fprintf(os.Stderr, "  • Known hosts file: %s\n", knownHostsFile)

	// Create or check known_hosts file
	knownHostsExists := true
	if _, err := os.Stat(knownHostsFile); err != nil {
		if os.IsNotExist(err) {
			knownHostsExists = false
			// Create the .ssh directory if it doesn't exist
			sshDir := filepath.Dir(knownHostsFile)
			if err := os.MkdirAll(sshDir, DirPermissionsSSH); err != nil {
				return nil, fmt.Errorf("failed to create .ssh directory: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to access known_hosts file %s: %w", knownHostsFile, err)
		}
	}

	// Create the host key callback using knownhosts if file exists
	var callback ssh.HostKeyCallback
	if knownHostsExists {
		var err error
		callback, err = knownhosts.New(knownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load known_hosts file %s: %w", knownHostsFile, err)
		}
	}

	// Wrap the callback to provide interactive acceptance and better error messages
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// If known_hosts doesn't exist, auto-accept if accept-new mode or prompt otherwise
		if !knownHostsExists {
			if acceptNew {
				return autoAcceptHostKey(hostname, key, knownHostsFile)
			}
			return promptToAcceptHostKey(hostname, key, knownHostsFile)
		}

		// Try to verify with known_hosts
		err := callback(hostname, remote, key)
		if err != nil {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				if len(keyErr.Want) == 0 {
					// Host key not found in known_hosts
					if acceptNew {
						return autoAcceptHostKey(hostname, key, knownHostsFile)
					}
					return promptToAcceptHostKey(hostname, key, knownHostsFile)
				}
				// Host key mismatch (potential MITM attack)
				return fmt.Errorf("host key verification failed: host key mismatch for %s\n"+
					"This could indicate a man-in-the-middle attack!\n"+
					"Expected key fingerprint to match entry in %s\n"+
					"To resolve: remove the old key and reconnect",
					hostname, knownHostsFile)
			}
			return fmt.Errorf("host key verification failed: %w", err)
		}
		return nil
	}, nil
}

// autoAcceptHostKey automatically accepts and saves an unknown host key without prompting
func autoAcceptHostKey(hostname string, key ssh.PublicKey, knownHostsFile string) error {
	fingerprint := ssh.FingerprintSHA256(key)
	keyType := key.Type()

	// Add the key to known_hosts file
	if err := addHostKeyToKnownHosts(hostname, key, knownHostsFile); err != nil {
		return fmt.Errorf("failed to add host key to known_hosts: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Permanently added '%s' (%s) to the list of known hosts.\n", hostname, keyType)
	fmt.Fprintf(os.Stderr, "  Fingerprint: %s\n", fingerprint)
	return nil
}

// promptToAcceptHostKey prompts the user to accept an unknown host key
func promptToAcceptHostKey(hostname string, key ssh.PublicKey, knownHostsFile string) error {
	fingerprint := ssh.FingerprintSHA256(key)
	keyType := key.Type()

	fmt.Fprintf(os.Stderr, "\n⚠️  The authenticity of host '%s' can't be established.\n", hostname)
	fmt.Fprintf(os.Stderr, "%s key fingerprint is %s.\n", keyType, fingerprint)
	fmt.Fprintf(os.Stderr, "Are you sure you want to continue connecting (yes/no)? ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "yes" {
		return fmt.Errorf("host key verification failed: connection rejected by user")
	}

	// Add the key to known_hosts file
	if err := addHostKeyToKnownHosts(hostname, key, knownHostsFile); err != nil {
		return fmt.Errorf("failed to add host key to known_hosts: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✓ Warning: Permanently added '%s' (%s) to the list of known hosts.\n\n", hostname, keyType)
	return nil
}

// addHostKeyToKnownHosts appends a host key to the known_hosts file
func addHostKeyToKnownHosts(hostname string, key ssh.PublicKey, knownHostsFile string) error {
	// Format: hostname key-type base64-key
	line := knownhosts.Line([]string{hostname}, key)

	// Open file in append mode, create if it doesn't exist
	// #nosec G304 -- knownHostsFile path comes from trusted sources (user config or default ~/.ssh/known_hosts)
	file, err := os.OpenFile(knownHostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, FilePermissionsKnownHosts)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}

	return nil
}

// createFingerprintCallback creates a callback that verifies against a specific fingerprint
func createFingerprintCallback(fingerprint, host string, port int) (ssh.HostKeyCallback, error) {
	// Normalize fingerprint (remove any SHA256: prefix and spaces)
	fingerprint = strings.TrimPrefix(fingerprint, "SHA256:")
	fingerprint = strings.ReplaceAll(fingerprint, " ", "")

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Calculate the fingerprint of the provided key
		actualFingerprint := ssh.FingerprintSHA256(key)
		expectedFingerprint := "SHA256:" + fingerprint

		if actualFingerprint != expectedFingerprint {
			return fmt.Errorf("host key fingerprint mismatch for %s\n"+
				"Expected: %s\n"+
				"Got:      %s\n"+
				"This could indicate a man-in-the-middle attack!",
				hostname, expectedFingerprint, actualFingerprint)
		}
		return nil
	}, nil
}
