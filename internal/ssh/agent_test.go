package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// startFakeAgent starts an in-process SSH agent (golang.org/x/crypto/ssh/agent.Keyring)
// listening on a unix socket, points SSH_AUTH_SOCK at it for the duration of the test,
// and returns the socket path. The listener and any accepted connections are cleaned
// up via t.Cleanup.
func startFakeAgent(t *testing.T, keys ...agent.AddedKey) string {
	t.Helper()

	keyring := agent.NewKeyring()
	for _, k := range keys {
		if err := keyring.Add(k); err != nil {
			t.Fatalf("failed to add key to fake agent: %v", err)
		}
	}

	// Use a short-lived dir outside t.TempDir(): the latter embeds the full
	// (sub)test name, which regularly exceeds the ~104 byte sockaddr_un limit
	// on macOS/BSD once nested subtests are involved.
	sockDir, err := os.MkdirTemp("", "shippy-agent")
	if err != nil {
		t.Fatalf("failed to create temp dir for agent socket: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(sockDir) })

	sockPath := filepath.Join(sockDir, "a.sock")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to start fake ssh-agent listener: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				_ = agent.ServeAgent(keyring, conn)
				_ = conn.Close()
			}()
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()
	})

	t.Setenv("SSH_AUTH_SOCK", sockPath)
	return sockPath
}

// generateEd25519Key returns a freshly generated ed25519 key pair.
func generateEd25519Key(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	return priv
}

// writeUnencryptedKeyFile writes an unencrypted PEM-encoded private key to a file
// under a fresh temp directory and returns its path.
func writeUnencryptedKeyFile(t *testing.T, key ed25519.PrivateKey) string {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(key, "test-key")
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	return writePEMKeyFile(t, block)
}

// writePassphraseProtectedKeyFile writes a passphrase-encrypted PEM-encoded private
// key to a file under a fresh temp directory and returns its path.
func writePassphraseProtectedKeyFile(t *testing.T, key ed25519.PrivateKey, passphrase string) string {
	t.Helper()
	block, err := ssh.MarshalPrivateKeyWithPassphrase(key, "test-key", []byte(passphrase))
	if err != nil {
		t.Fatalf("failed to marshal passphrase-protected private key: %v", err)
	}
	return writePEMKeyFile(t, block)
}

func writePEMKeyFile(t *testing.T, block *pem.Block) string {
	t.Helper()
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "id_ed25519")
	f, err := os.OpenFile(keyPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	defer f.Close()
	if err := pem.Encode(f, block); err != nil {
		t.Fatalf("failed to write PEM key: %v", err)
	}
	return keyPath
}

// emptyHomeDir points $HOME at a fresh, key-less temp directory so default SSH key
// discovery (~/.ssh/id_ed25519 etc.) reliably finds nothing, independent of the
// machine actually running the test.
func emptyHomeDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// TestNewClientWithOptions_AgentOnly_NoKeyFileNeeded is also the regression test for
// the exact bug reported against v0.0.9: with no ssh_key configured in .shippy.yaml
// and no ~/.ssh/config entry, github.com/kevinburke/ssh_config.Get still returns its
// own hardcoded "~/.ssh/identity" default for IdentityFile. That must never be
// treated as if the user explicitly configured a key - if an agent is available, a
// missing ~/.ssh/identity must not cause a hard failure.
func TestNewClientWithOptions_AgentOnly_NoKeyFileNeeded(t *testing.T) {
	emptyHomeDir(t)
	startFakeAgent(t, agent.AddedKey{PrivateKey: generateEd25519Key(t)})

	client, err := NewClientWithOptions(ClientOptions{
		Host: "example.invalid",
		Port: 22,
		User: "deploy",
	})
	if err != nil {
		t.Fatalf("expected agent-only auth to succeed without ssh_key, got error: %v", err)
	}
	defer client.Close()

	if len(client.config.Auth) != 1 {
		t.Fatalf("expected exactly 1 auth method (agent), got %d", len(client.config.Auth))
	}
	if client.agentConn == nil {
		t.Fatal("expected agentConn to be set when ssh-agent is available")
	}
}

func TestNewClientWithOptions_NoKeyNoAgent_Fails(t *testing.T) {
	emptyHomeDir(t)
	t.Setenv("SSH_AUTH_SOCK", "")

	_, err := NewClientWithOptions(ClientOptions{
		Host: "example.invalid",
		Port: 22,
		User: "deploy",
	})
	if err == nil {
		t.Fatal("expected error when neither ssh_key nor ssh-agent is available")
	}
	// Note: opts.KeyPath is empty here, but applySSHConfig() (via
	// github.com/kevinburke/ssh_config) fills in its own hardcoded
	// ~/.ssh/identity default in the absence of a real ~/.ssh/config entry, so
	// this doesn't hit the "no default SSH key found in ~/.ssh/" message -
	// it falls through to file-not-found -> no usable auth method at all.
	if !strings.Contains(err.Error(), "no SSH authentication method available") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewClientWithOptions_PassphraseKeyWithAgent_SkipsFileKeyInsteadOfFailing(t *testing.T) {
	emptyHomeDir(t)
	startFakeAgent(t, agent.AddedKey{PrivateKey: generateEd25519Key(t)})

	keyPath := writePassphraseProtectedKeyFile(t, generateEd25519Key(t), "correct horse battery staple")

	client, err := NewClientWithOptions(ClientOptions{
		Host:    "example.invalid",
		Port:    22,
		User:    "deploy",
		KeyPath: keyPath,
	})
	if err != nil {
		t.Fatalf("expected passphrase-protected key + available agent to succeed, got error: %v", err)
	}
	defer client.Close()

	// Only the agent-backed auth method should have been added; the encrypted
	// file key must be skipped rather than causing a hard failure.
	if len(client.config.Auth) != 1 {
		t.Fatalf("expected exactly 1 auth method (agent only, file key skipped), got %d", len(client.config.Auth))
	}
}

func TestNewClientWithOptions_PassphraseKeyWithoutAgent_FailsWithHelpfulError(t *testing.T) {
	emptyHomeDir(t)
	t.Setenv("SSH_AUTH_SOCK", "")

	keyPath := writePassphraseProtectedKeyFile(t, generateEd25519Key(t), "correct horse battery staple")

	_, err := NewClientWithOptions(ClientOptions{
		Host:    "example.invalid",
		Port:    22,
		User:    "deploy",
		KeyPath: keyPath,
	})
	if err == nil {
		t.Fatal("expected error when key is passphrase-protected and no agent is available")
	}
	if !strings.Contains(err.Error(), "passphrase-protected") || !strings.Contains(err.Error(), "ssh-add") {
		t.Fatalf("expected error to mention passphrase protection and ssh-add remedy, got: %v", err)
	}
}

func TestNewClientWithOptions_ExplicitMissingKeyPath_FailsEvenWithAgent(t *testing.T) {
	emptyHomeDir(t)
	startFakeAgent(t, agent.AddedKey{PrivateKey: generateEd25519Key(t)})

	_, err := NewClientWithOptions(ClientOptions{
		Host:    "example.invalid",
		Port:    22,
		User:    "deploy",
		KeyPath: filepath.Join(t.TempDir(), "does-not-exist"),
	})
	if err == nil {
		t.Fatal("expected explicit but missing ssh_key to still be a hard error, even with an agent available")
	}
	if !strings.Contains(err.Error(), "SSH key not found") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestNewClientWithOptions_FileKeyAndAgent_BothOffered(t *testing.T) {
	emptyHomeDir(t)
	startFakeAgent(t, agent.AddedKey{PrivateKey: generateEd25519Key(t)})

	keyPath := writeUnencryptedKeyFile(t, generateEd25519Key(t))

	client, err := NewClientWithOptions(ClientOptions{
		Host:    "example.invalid",
		Port:    22,
		User:    "deploy",
		KeyPath: keyPath,
	})
	if err != nil {
		t.Fatalf("expected success with both a valid file key and an agent, got error: %v", err)
	}
	defer client.Close()

	// Agent signers and the file key must be combined into a SINGLE publickey
	// AuthMethod. Registering them as two separate methods regresses to the bug
	// where x/crypto/ssh only ever tries the first publickey method, so an empty
	// or wrong-key agent prevents the configured ssh_key from being offered.
	if len(client.config.Auth) != 1 {
		t.Fatalf("expected agent and file key to be combined into 1 auth method, got %d", len(client.config.Auth))
	}
}

func TestNewClientWithOptions_FileKeyOnly_NoAgent_StillWorks(t *testing.T) {
	emptyHomeDir(t)
	t.Setenv("SSH_AUTH_SOCK", "")

	keyPath := writeUnencryptedKeyFile(t, generateEd25519Key(t))

	client, err := NewClientWithOptions(ClientOptions{
		Host:    "example.invalid",
		Port:    22,
		User:    "deploy",
		KeyPath: keyPath,
	})
	if err != nil {
		t.Fatalf("expected unchanged pre-existing behavior (file key, no agent) to still work, got error: %v", err)
	}
	defer client.Close()

	if len(client.config.Auth) != 1 {
		t.Fatalf("expected exactly 1 auth method (file key), got %d", len(client.config.Auth))
	}
	if client.agentConn != nil {
		t.Fatal("expected agentConn to be nil when no ssh-agent is available")
	}
}

// startSSHServer stands up an in-process SSH server on 127.0.0.1 that accepts
// publickey auth only for authorizedKey. It returns the host and port to dial.
// Accepted connections complete the handshake and then discard all channels and
// global requests, which is enough for NewClient/Connect to succeed.
var errKeyNotAuthorized = errors.New("key not authorized")

func startSSHServer(t *testing.T, authorizedKey ssh.PublicKey) (string, int) {
	t.Helper()

	hostSigner, err := ssh.NewSignerFromKey(generateEd25519Key(t))
	if err != nil {
		t.Fatalf("failed to build host signer: %v", err)
	}

	authorized := authorizedKey.Marshal()
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(_ ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), authorized) {
				return &ssh.Permissions{}, nil
			}
			return nil, errKeyNotAuthorized
		},
	}
	config.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start SSH server listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
				if err != nil {
					_ = conn.Close()
					return
				}
				go ssh.DiscardRequests(reqs)
				go func() {
					for newChan := range chans {
						_ = newChan.Reject(ssh.Prohibited, "no channels in test server")
					}
				}()
				_ = sshConn.Wait()
			}()
		}
	}()

	addr := listener.Addr().(*net.TCPAddr)
	return "127.0.0.1", addr.Port
}

// TestNewClientWithOptions_EmptyAgentDoesNotShadowFileKey is the end-to-end
// regression test for the reported v0.1.2 failure: with an ssh-agent available
// but holding NO keys, the configured ssh_key must still authenticate. Before the
// fix the agent and file key were two separate publickey AuthMethods, and
// x/crypto/ssh only ever tries the first one - so the empty agent silently
// consumed the single publickey attempt and the file key was never offered,
// producing "attempted methods [none publickey], no supported methods remain".
func TestNewClientWithOptions_EmptyAgentDoesNotShadowFileKey(t *testing.T) {
	emptyHomeDir(t)
	// Agent is reachable but empty (macOS default: keys aren't auto-loaded).
	startFakeAgent(t)

	clientKey := generateEd25519Key(t)
	keyPath := writeUnencryptedKeyFile(t, clientKey)

	clientPub, err := ssh.NewPublicKey(clientKey.Public())
	if err != nil {
		t.Fatalf("failed to derive client public key: %v", err)
	}
	host, port := startSSHServer(t, clientPub)

	client, err := NewClientWithOptions(ClientOptions{
		Host:       host,
		Port:       port,
		User:       "deploy",
		KeyPath:    keyPath,
		SSHOptions: map[string]string{"StrictHostKeyChecking": "no"},
	})
	if err != nil {
		t.Fatalf("client construction failed: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		t.Fatalf("expected file-key auth to succeed despite empty agent, got: %v", err)
	}
}

func TestClientClose_ClosesAgentConn(t *testing.T) {
	emptyHomeDir(t)
	startFakeAgent(t, agent.AddedKey{PrivateKey: generateEd25519Key(t)})

	client, err := NewClientWithOptions(ClientOptions{
		Host: "example.invalid",
		Port: 22,
		User: "deploy",
	})
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	if client.agentConn == nil {
		t.Fatal("expected agentConn to be set")
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}
	if client.agentConn != nil {
		t.Fatal("expected agentConn to be cleared after Close")
	}
}
