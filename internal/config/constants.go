package config

// Deployment configuration defaults
const (
	// DefaultKeepReleases is the default number of releases to keep on the server
	DefaultKeepReleases = 5

	// DefaultLockTimeoutMinutes is the default deployment lock timeout in minutes
	// This matches Capistrano's default behavior
	DefaultLockTimeoutMinutes = 15

	// DefaultFileMode is the octal mode applied to deployed files. The exec bit is
	// additionally preserved for files that carry it in the source (see the rsync
	// --chmod handling in internal/rsync).
	DefaultFileMode = "0644"

	// DefaultDirMode is the octal mode applied to deployed directories. It includes
	// the setgid bit (2xxx) so files created under a release inherit the group; the
	// bit is realized via filesystem inheritance from the release tree.
	DefaultDirMode = "2755"
)
