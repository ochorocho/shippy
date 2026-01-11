package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	// Composer.json path (can be relative to config file or absolute)
	Composer string `yaml:"composer,omitempty"`

	// Global settings (can be overridden per-host)
	RsyncSrc     string   `yaml:"rsync_src,omitempty"`
	Exclude      []string `yaml:"exclude,omitempty"`
	Include      []string `yaml:"include,omitempty"`
	Shared       []string `yaml:"shared,omitempty"`
	KeepReleases int      `yaml:"keep_releases,omitempty"`
	LockEnabled  *bool    `yaml:"lock_enabled,omitempty"`
	LockTimeout  int      `yaml:"lock_timeout,omitempty"`

	// Host and command configuration
	Hosts    map[string]Host `yaml:"hosts"`
	Commands []Command       `yaml:"commands"`

	// Internal: path to config file (used for resolving relative paths)
	configPath string
}

// Host represents a deployment target
type Host struct {
	Hostname        string            `yaml:"hostname"`
	Port            int               `yaml:"port,omitempty"`
	RemoteUser      string            `yaml:"remote_user"`
	DeployPath      string            `yaml:"deploy_path"`
	RsyncSrc        string            `yaml:"rsync_src"`
	SSHKey          string            `yaml:"ssh_key,omitempty"`
	SSHOptions      map[string]string `yaml:"ssh_options,omitempty"`
	SSHMultiplexing bool              `yaml:"ssh_multiplexing,omitempty"`
	Exclude         []string          `yaml:"exclude,omitempty"`
	Include         []string          `yaml:"include,omitempty"`
	Shared          []string          `yaml:"shared,omitempty"`
	KeepReleases    int               `yaml:"keep_releases,omitempty"`
	LockEnabled     *bool             `yaml:"lock_enabled,omitempty"`  // Pointer to distinguish unset from false
	LockTimeout     int               `yaml:"lock_timeout,omitempty"`  // Timeout in minutes
}

// Command represents a command to execute
type Command struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Store absolute path to config file for resolving relative paths
	absPath, err := filepath.Abs(path)
	if err != nil {
		// Fall back to original path if we can't get absolute path
		cfg.configPath = path
	} else {
		cfg.configPath = absPath
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Hosts) == 0 {
		return fmt.Errorf("no hosts defined in configuration")
	}

	for name, host := range c.Hosts {
		if host.Hostname == "" {
			return fmt.Errorf("host '%s': hostname is required", name)
		}
		if host.RemoteUser == "" {
			return fmt.Errorf("host '%s': remote_user is required", name)
		}
		if host.DeployPath == "" {
			return fmt.Errorf("host '%s': deploy_path is required", name)
		}
		// rsync_src is now optional (has global default of ".")
	}

	return nil
}

// GetHost retrieves a specific host configuration
func (c *Config) GetHost(name string) (*Host, error) {
	host, ok := c.Hosts[name]
	if !ok {
		return nil, fmt.Errorf("host '%s' not found in configuration", name)
	}
	return &host, nil
}

// IsLockEnabled returns whether deployment locking is enabled (default: true)
func (h *Host) IsLockEnabled() bool {
	if h.LockEnabled == nil {
		return true // Default to enabled
	}
	return *h.LockEnabled
}

// GetLockTimeout returns the lock timeout in minutes (default: 15)
func (h *Host) GetLockTimeout() int {
	if h.LockTimeout <= 0 {
		return 15 // Default 15 minutes like Capistrano
	}
	return h.LockTimeout
}

// GetRsyncSrc returns the rsync source path (per-host override, global, or default)
func (c *Config) GetRsyncSrc(host *Host) string {
	if host.RsyncSrc != "" {
		return host.RsyncSrc // Per-host override
	}
	if c.RsyncSrc != "" {
		return c.RsyncSrc // Global setting
	}
	return "." // Default to current directory
}

// GetExclude returns exclude patterns (per-host override or global default)
func (c *Config) GetExclude(host *Host) []string {
	if len(host.Exclude) > 0 {
		return host.Exclude // Per-host override
	}
	return c.Exclude // Global default
}

// GetInclude returns include patterns (per-host override or global default)
func (c *Config) GetInclude(host *Host) []string {
	if len(host.Include) > 0 {
		return host.Include // Per-host override
	}
	return c.Include // Global default
}

// GetShared returns shared files/directories (per-host override or global default)
func (c *Config) GetShared(host *Host) []string {
	if len(host.Shared) > 0 {
		return host.Shared // Per-host override
	}
	return c.Shared // Global default
}

// GetKeepReleases returns keep_releases value (per-host override, global, or default)
func (c *Config) GetKeepReleases(host *Host) int {
	if host.KeepReleases > 0 {
		return host.KeepReleases // Per-host override
	}
	if c.KeepReleases > 0 {
		return c.KeepReleases // Global setting
	}
	return 5 // Default
}

// IsLockEnabled returns whether locking is enabled (per-host override, global, or default)
func (c *Config) IsLockEnabled(host *Host) bool {
	if host.LockEnabled != nil {
		return *host.LockEnabled // Per-host override
	}
	if c.LockEnabled != nil {
		return *c.LockEnabled // Global setting
	}
	return true // Default enabled
}

// GetLockTimeout returns lock timeout (per-host override, global, or default)
func (c *Config) GetLockTimeout(host *Host) int {
	if host.LockTimeout > 0 {
		return host.LockTimeout // Per-host override
	}
	if c.LockTimeout > 0 {
		return c.LockTimeout // Global setting
	}
	return 15 // Default 15 minutes
}

// ApplyDefaults sets default values for unspecified fields
func (c *Config) ApplyDefaults() {
	// Set global defaults if not specified
	if c.RsyncSrc == "" {
		c.RsyncSrc = "."
	}
	if c.KeepReleases == 0 {
		c.KeepReleases = 5
	}
	if c.LockEnabled == nil {
		enabled := true
		c.LockEnabled = &enabled
	}
	if c.LockTimeout == 0 {
		c.LockTimeout = 15
	}

	// Set default TYPO3 commands if not specified
	if len(c.Commands) == 0 {
		c.Commands = []Command{
			{
				Name: "Run extension setup",
				Run:  "./{{config.bin-dir|vendor/bin}}/typo3 extension:setup",
			},
			{
				Name: "Run upgrade wizards",
				Run:  "./{{config.bin-dir|vendor/bin}}/typo3 upgrade:run",
			},
			{
				Name: "Update language files",
				Run:  "./{{config.bin-dir|vendor/bin}}/typo3 language:update",
			},
			{
				Name: "Flush caches",
				Run:  "./{{config.bin-dir|vendor/bin}}/typo3 cache:flush",
			},
			{
				Name: "Warmup caches",
				Run:  "./{{config.bin-dir|vendor/bin}}/typo3 cache:warmup",
			},
		}
	}
}

// GetComposerPath returns the resolved composer.json path
// Supports both relative (to config file) and absolute paths
func (c *Config) GetComposerPath() string {
	// If not specified, use default
	if c.Composer == "" {
		return "composer.json"
	}

	// If absolute path, return as-is
	if filepath.IsAbs(c.Composer) {
		return c.Composer
	}

	// If relative path and we have a config path, resolve relative to config file directory
	if c.configPath != "" {
		configDir := filepath.Dir(c.configPath)
		return filepath.Join(configDir, c.Composer)
	}

	// Fallback: return as-is (relative to CWD)
	return c.Composer
}
