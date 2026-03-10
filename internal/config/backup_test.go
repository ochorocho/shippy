package config

import (
	"testing"
)

func TestBackupDatabaseConfig_HasManualCredentials(t *testing.T) {
	tests := []struct {
		name   string
		config BackupDatabaseConfig
		want   bool
	}{
		{
			name:   "all fields set",
			config: BackupDatabaseConfig{Driver: "mysql", Host: "localhost", Name: "mydb", User: "root"},
			want:   true,
		},
		{
			name:   "missing driver",
			config: BackupDatabaseConfig{Host: "localhost", Name: "mydb", User: "root"},
			want:   false,
		},
		{
			name:   "missing host",
			config: BackupDatabaseConfig{Driver: "mysql", Name: "mydb", User: "root"},
			want:   false,
		},
		{
			name:   "missing name",
			config: BackupDatabaseConfig{Driver: "mysql", Host: "localhost", User: "root"},
			want:   false,
		},
		{
			name:   "missing user",
			config: BackupDatabaseConfig{Driver: "mysql", Host: "localhost", Name: "mydb"},
			want:   false,
		},
		{
			name:   "all empty",
			config: BackupDatabaseConfig{},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.HasManualCredentials()
			if got != tt.want {
				t.Errorf("HasManualCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackupDatabaseConfig_GetCredentialSource(t *testing.T) {
	tests := []struct {
		name   string
		config BackupDatabaseConfig
		want   string
	}{
		{
			name:   "explicit credential source",
			config: BackupDatabaseConfig{Credentials: "typo3"},
			want:   "typo3",
		},
		{
			name:   "manual when all fields set",
			config: BackupDatabaseConfig{Driver: "mysql", Host: "localhost", Name: "mydb", User: "root"},
			want:   "manual",
		},
		{
			name:   "auto when nothing set",
			config: BackupDatabaseConfig{},
			want:   "auto",
		},
		{
			name:   "auto when partial fields set",
			config: BackupDatabaseConfig{Driver: "mysql"},
			want:   "auto",
		},
		{
			name:   "explicit overrides manual detection",
			config: BackupDatabaseConfig{Credentials: "dotenv", Driver: "mysql", Host: "localhost", Name: "mydb", User: "root"},
			want:   "dotenv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetCredentialSource()
			if got != tt.want {
				t.Errorf("GetCredentialSource() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_GetBackup(t *testing.T) {
	globalBackup := &BackupConfig{Output: "./global"}
	hostBackup := &BackupConfig{Output: "./host"}

	cfg := &Config{
		Backup: globalBackup,
	}

	// Host without override returns global
	host := &Host{}
	got := cfg.GetBackup(host)
	if got != globalBackup {
		t.Error("expected global backup config when host has no override")
	}

	// Host with override returns host-level
	host = &Host{Backup: hostBackup}
	got = cfg.GetBackup(host)
	if got != hostBackup {
		t.Error("expected host backup config when host has override")
	}

	// No global config returns nil
	cfg = &Config{}
	host = &Host{}
	got = cfg.GetBackup(host)
	if got != nil {
		t.Error("expected nil when no backup config exists")
	}
}

func TestConfig_GetBackupOutput(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		host   *Host
		want   string
	}{
		{
			name:   "default when no backup config",
			config: &Config{},
			host:   &Host{},
			want:   ".",
		},
		{
			name:   "global output",
			config: &Config{Backup: &BackupConfig{Output: "./backups"}},
			host:   &Host{},
			want:   "./backups",
		},
		{
			name:   "host override",
			config: &Config{Backup: &BackupConfig{Output: "./global"}},
			host:   &Host{Backup: &BackupConfig{Output: "./host-backups"}},
			want:   "./host-backups",
		},
		{
			name:   "empty output falls back to default",
			config: &Config{Backup: &BackupConfig{}},
			host:   &Host{},
			want:   ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetBackupOutput(tt.host)
			if got != tt.want {
				t.Errorf("GetBackupOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}
