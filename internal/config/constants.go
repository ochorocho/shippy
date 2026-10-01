package config

// Deployment configuration defaults
const (
	// DefaultKeepReleases is the default number of releases to keep on the server
	DefaultKeepReleases = 5

	// DefaultLockTimeoutMinutes is the default deployment lock timeout in minutes
	// This matches Capistrano's default behavior
	DefaultLockTimeoutMinutes = 15

	// DefaultFileMode is the octal mode applied to deployed files. It is
	// group-writable so a group member (e.g. PHP running as www-data) can modify
	// deployed files; the exec bit is additionally preserved for files that carry it
	// in the source (see the rsync --chmod handling in internal/rsync).
	DefaultFileMode = "0664"

	// DefaultDirMode is the octal mode applied to deployed directories:
	// group-writable with the setgid bit (2xxx) so new files inherit the parent
	// group. --chmod sets the base 2775 bits; the actual setgid "s" flag and the
	// group (e.g. www-data) are realized via filesystem inheritance from the setgid
	// deploy tree (.cache/ and releases/ created under a setgid $siteroot).
	DefaultDirMode = "2775"
)
