package backup

import (
	"fmt"
	"io"
	"path/filepath"

	"shippy/internal/ssh"
)

// Dumper defines the interface for database-specific dump implementations
type Dumper interface {
	Dump(w io.Writer, excludeTables []string, options map[string]string) error
	Close() error
}

// DatabaseCredentials holds parsed database connection info
type DatabaseCredentials struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// NewDumper creates the appropriate dumper based on the database driver.
// It opens an SSH tunnel to the remote database and connects through it.
func NewDumper(client *ssh.Client, creds *DatabaseCredentials) (Dumper, error) {
	switch creds.Driver {
	case "mysql", "mariadb", "mysqli", "pdo_mysql":
		return NewMySQLDumper(client, creds)
	case "postgresql", "postgres", "pgsql", "pdo_pgsql":
		return NewPostgresDumper(client, creds)
	case "sqlite", "sqlite3", "pdo_sqlite":
		return NewSQLiteDumper(client, creds)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", creds.Driver)
	}
}

// MatchesExcludePattern checks if a table name matches any of the exclude patterns.
// Supports glob patterns like "cache_*", "cf_*", "sys_log".
func MatchesExcludePattern(tableName string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, tableName)
		if err == nil && matched {
			return true
		}
	}
	return false
}
