package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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

	// CommandContext wraps every post-deploy/rollback command so it runs inside
	// a subcontext (e.g. a container). When set, commands are executed as
	// `<command_context> sh -c '<cd releasePath && run>'`. Empty = run directly
	// on the remote host (default). Can be overridden per-host and per-command.
	CommandContext string `yaml:"command_context,omitempty"`

	// Backup configuration
	Backup *BackupConfig `yaml:"backup,omitempty"`

	// Host and command configuration
	Hosts            map[string]Host `yaml:"hosts"`
	Commands         []Command       `yaml:"commands"`
	RollbackCommands []Command       `yaml:"rollback_commands,omitempty"`

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
	LockEnabled     *bool             `yaml:"lock_enabled,omitempty"`    // Pointer to distinguish unset from false
	LockTimeout     int               `yaml:"lock_timeout,omitempty"`    // Timeout in minutes
	Backup          *BackupConfig     `yaml:"backup,omitempty"`          // Per-host backup override
	CommandContext  string            `yaml:"command_context,omitempty"` // Per-host command context override
}

// Command represents a command to execute
type Command struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`

	// CommandContext overrides the host/global command context for this single
	// command. A pointer so an explicit empty string ("") can force the command
	// to run directly on the host even when a host/global context is set; nil
	// means "inherit the host/global context".
	CommandContext *string `yaml:"command_context,omitempty"`

	// Only restricts this command to the listed host names (the keys under
	// `hosts:`). Empty = runs on every host unless excluded below.
	Only []string `yaml:"only,omitempty"`

	// Except skips this command on the listed host names, even if Only would
	// otherwise include it.
	Except []string `yaml:"except,omitempty"`
}

// AppliesToHost reports whether the command should run when deploying to
// hostName, per its only/except filters (GitLab CI style).
func (cmd Command) AppliesToHost(hostName string) bool {
	if len(cmd.Only) > 0 && !slices.Contains(cmd.Only, hostName) {
		return false
	}
	if slices.Contains(cmd.Except, hostName) {
		return false
	}
	return true
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	// #nosec G304 -- Config path comes from CLI flag or default .shippy.yaml
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

	// Global shared entries are reused across all hosts.
	for i, shared := range c.Shared {
		if err := validateShellSafe(shared, fmt.Sprintf("shared[%d]", i)); err != nil {
			return err
		}
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

		// These values are interpolated into remote shell commands (always via
		// ssh.Quote). Reject control characters that can never appear in a
		// legitimate path/host and would only indicate a corrupt config or an
		// injection attempt.
		if err := validateShellSafe(host.Hostname, fmt.Sprintf("host '%s': hostname", name)); err != nil {
			return err
		}
		if err := validateShellSafe(host.RemoteUser, fmt.Sprintf("host '%s': remote_user", name)); err != nil {
			return err
		}
		if err := validateShellSafe(host.DeployPath, fmt.Sprintf("host '%s': deploy_path", name)); err != nil {
			return err
		}
		if err := validateShellSafe(host.SSHKey, fmt.Sprintf("host '%s': ssh_key", name)); err != nil {
			return err
		}
		if err := validateShellSafe(host.CommandContext, fmt.Sprintf("host '%s': command_context", name)); err != nil {
			return err
		}
		for i, shared := range host.Shared {
			if err := validateShellSafe(shared, fmt.Sprintf("host '%s': shared[%d]", name, i)); err != nil {
				return err
			}
		}
	}

	if err := validateShellSafe(c.CommandContext, "command_context"); err != nil {
		return err
	}
	for i, cmd := range c.Commands {
		if cmd.CommandContext != nil {
			if err := validateShellSafe(*cmd.CommandContext, fmt.Sprintf("commands[%d]: command_context", i)); err != nil {
				return err
			}
		}
		if err := c.validateCommandHostRefs(cmd, fmt.Sprintf("commands[%d]", i)); err != nil {
			return err
		}
	}
	for i, cmd := range c.RollbackCommands {
		if cmd.CommandContext != nil {
			if err := validateShellSafe(*cmd.CommandContext, fmt.Sprintf("rollback_commands[%d]: command_context", i)); err != nil {
				return err
			}
		}
		if err := c.validateCommandHostRefs(cmd, fmt.Sprintf("rollback_commands[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

// validateCommandHostRefs checks that a command's only/except entries refer
// to hosts actually defined in the config, catching typos at load time.
func (c *Config) validateCommandHostRefs(cmd Command, field string) error {
	for _, hostName := range cmd.Only {
		if _, ok := c.Hosts[hostName]; !ok {
			return fmt.Errorf("%s: only references unknown host '%s'", field, hostName)
		}
	}
	for _, hostName := range cmd.Except {
		if _, ok := c.Hosts[hostName]; !ok {
			return fmt.Errorf("%s: except references unknown host '%s'", field, hostName)
		}
	}
	return nil
}

// validateShellSafe rejects values containing control characters (NUL, newline,
// carriage return). All such values are passed through ssh.Quote before being
// sent to the remote shell, so this is defense-in-depth: a newline or NUL in a
// path indicates a malformed config or a deliberate injection attempt and is
// never legitimate, so we fail loudly at load time rather than silently.
func validateShellSafe(value, field string) error {
	if strings.ContainsAny(value, "\x00\n\r") {
		return fmt.Errorf("%s contains illegal control characters (newline, carriage return or null byte)", field)
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
		return DefaultLockTimeoutMinutes // Default like Capistrano
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
	return DefaultKeepReleases // Default
}

// GetCommandContext resolves the command context for a single command using
// precedence: per-command override > per-host > global. A per-command context
// of "" (non-nil) explicitly forces execution on the host; nil inherits.
func (c *Config) GetCommandContext(host *Host, cmd Command) string {
	if cmd.CommandContext != nil {
		return *cmd.CommandContext // Explicit per-command override ("" = run on host)
	}
	if host != nil && host.CommandContext != "" {
		return host.CommandContext // Per-host override
	}
	return c.CommandContext // Global setting (may be empty)
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
	return DefaultLockTimeoutMinutes // Default
}

// ApplyDefaults sets default values for unspecified fields
func (c *Config) ApplyDefaults() {
	// Set global defaults if not specified
	if c.RsyncSrc == "" {
		c.RsyncSrc = "."
	}
	if c.KeepReleases == 0 {
		c.KeepReleases = DefaultKeepReleases
	}
	if c.LockEnabled == nil {
		enabled := true
		c.LockEnabled = &enabled
	}
	if c.LockTimeout == 0 {
		c.LockTimeout = DefaultLockTimeoutMinutes
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

// Redacted returns a copy of the config with secret values (e.g. backup
// database passwords) replaced with a placeholder. Safe to print or log,
// unlike marshalling the config directly.
func (c *Config) Redacted() *Config {
	redacted := *c

	if redacted.Backup != nil {
		redacted.Backup = redacted.Backup.redacted()
	}

	if redacted.Hosts != nil {
		hosts := make(map[string]Host, len(redacted.Hosts))
		for name, host := range redacted.Hosts {
			if host.Backup != nil {
				host.Backup = host.Backup.redacted()
			}
			hosts[name] = host
		}
		redacted.Hosts = hosts
	}

	return &redacted
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
