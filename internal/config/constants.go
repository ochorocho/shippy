package config

// Deployment configuration defaults
const (
	// DefaultKeepReleases is the default number of releases to keep on the server
	DefaultKeepReleases = 5

	// DefaultLockTimeoutMinutes is the default deployment lock timeout in minutes
	// This matches Capistrano's default behavior
	DefaultLockTimeoutMinutes = 15
)
