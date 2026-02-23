package config

// BackupConfig holds backup-related configuration
type BackupConfig struct {
	Database *BackupDatabaseConfig `yaml:"database,omitempty"`
	Output   string                `yaml:"output,omitempty"` // Local output directory (default: ".")
	Files    []string              `yaml:"files,omitempty"`  // Files/directories to backup from shared/
}

// BackupDatabaseConfig holds database backup settings
type BackupDatabaseConfig struct {
	// Credential source: "auto" (default), "dotenv", "typo3", "manual"
	Credentials string `yaml:"credentials,omitempty"`

	// Manual override fields (skip PHP extraction when all set)
	Driver   string `yaml:"driver,omitempty"`   // mysql, postgresql, sqlite
	Host     string `yaml:"host,omitempty"`     // Database host as seen from the remote server
	Port     int    `yaml:"port,omitempty"`     // Database port
	Name     string `yaml:"name,omitempty"`     // Database name
	User     string `yaml:"user,omitempty"`     // Database user
	Password string `yaml:"password,omitempty"` // Database password

	// Common settings
	ExcludeTables []string          `yaml:"exclude_tables,omitempty"` // Glob patterns: "cache_*", "cf_*"
	Options       map[string]string `yaml:"options,omitempty"`        // DBMS-specific options
}

// GetBackup returns backup config (per-host override or global default)
func (c *Config) GetBackup(host *Host) *BackupConfig {
	if host.Backup != nil {
		return host.Backup
	}
	return c.Backup
}

// GetBackupOutput returns the backup output directory (default: ".")
func (c *Config) GetBackupOutput(host *Host) string {
	backup := c.GetBackup(host)
	if backup != nil && backup.Output != "" {
		return backup.Output
	}
	return "."
}

// GetBackupDatabase returns the database backup config (per-host override or global)
func (c *Config) GetBackupDatabase(host *Host) *BackupDatabaseConfig {
	backup := c.GetBackup(host)
	if backup == nil {
		return nil
	}
	return backup.Database
}

// GetBackupFiles returns the list of files/directories to backup
func (c *Config) GetBackupFiles(host *Host) []string {
	backup := c.GetBackup(host)
	if backup == nil {
		return nil
	}
	return backup.Files
}

// HasManualCredentials returns true if all manual credential fields are set
func (d *BackupDatabaseConfig) HasManualCredentials() bool {
	return d.Driver != "" && d.Host != "" && d.Name != "" && d.User != ""
}

// GetCredentialSource returns the credential source (default: "auto")
func (d *BackupDatabaseConfig) GetCredentialSource() string {
	if d.Credentials != "" {
		return d.Credentials
	}
	if d.HasManualCredentials() {
		return "manual"
	}
	return "auto"
}
