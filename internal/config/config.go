package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Hosts    map[string]Host `yaml:"hosts"`
	Commands []Command       `yaml:"commands"`
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
		if host.RsyncSrc == "" {
			return fmt.Errorf("host '%s': rsync_src is required", name)
		}
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
