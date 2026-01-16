package ssh

import "time"

// SSH connection defaults
const (
	// DefaultSSHPort is the standard SSH port
	DefaultSSHPort = 22

	// DefaultConnectTimeout is the default timeout for SSH connection establishment
	DefaultConnectTimeout = 30 * time.Second
)

// File and directory permissions for SSH-related files
const (
	// DirPermissionsSSH is the permission mode for .ssh directory (rwx------)
	DirPermissionsSSH = 0700

	// FilePermissionsKnownHosts is the permission mode for known_hosts file (rw-------)
	FilePermissionsKnownHosts = 0600

	// FilePermissionsPrivateKey is the permission mode for SSH private keys (rw-------)
	FilePermissionsPrivateKey = 0600
)
