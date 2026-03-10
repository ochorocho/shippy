package backup

import (
	"fmt"
	"io"

	"shippy/internal/ssh"
)

// SQLiteDumper downloads a SQLite database file from the remote server
type SQLiteDumper struct {
	client *ssh.Client
	creds  *DatabaseCredentials
}

// NewSQLiteDumper creates a SQLite dumper that downloads the database file via SSH
func NewSQLiteDumper(client *ssh.Client, creds *DatabaseCredentials) (*SQLiteDumper, error) {
	if creds.Name == "" {
		return nil, fmt.Errorf("SQLite database path (name) is required")
	}
	return &SQLiteDumper{client: client, creds: creds}, nil
}

// Dump streams the SQLite database file to w.
// For SQLite, the dump is a raw copy of the database file since it's file-based.
// excludeTables and options are ignored for SQLite (the entire file is copied).
func (d *SQLiteDumper) Dump(w io.Writer, _ []string, _ map[string]string) error {
	// Stream the file via SSH cat command
	output, err := d.client.RunCommand(fmt.Sprintf("cat %s", d.creds.Name))
	if err != nil {
		return fmt.Errorf("failed to read SQLite database '%s': %w", d.creds.Name, err)
	}

	if _, err := io.WriteString(w, output); err != nil {
		return fmt.Errorf("failed to write SQLite database: %w", err)
	}

	return nil
}

// Close is a no-op for SQLite since there's no persistent connection
func (d *SQLiteDumper) Close() error {
	return nil
}
