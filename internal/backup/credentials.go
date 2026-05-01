package backup

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"shippy/internal/config"
	"shippy/internal/ssh"
)

//go:embed credentials.php
var credentialsScript embed.FS

// ExtractCredentials extracts database credentials from the remote server.
// It first checks for manual config, then uses a PHP script to read
// the application's configuration files.
func ExtractCredentials(client *ssh.Client, deployPath string, dbConfig *config.BackupDatabaseConfig) (*DatabaseCredentials, error) {
	// If manual credentials are fully specified, use them directly
	if dbConfig.HasManualCredentials() {
		return &DatabaseCredentials{
			Driver:   normalizeDriver(dbConfig.Driver),
			Host:     dbConfig.Host,
			Port:     dbConfig.Port,
			Name:     dbConfig.Name,
			User:     dbConfig.User,
			Password: dbConfig.Password,
		}, nil
	}

	// Extract credentials using PHP script on the remote server
	creds, err := extractViaPhp(client, deployPath, dbConfig.GetCredentialSource())
	if err != nil {
		// If PHP extraction fails and we have partial manual config, try that
		if dbConfig.Driver != "" && dbConfig.Name != "" {
			return &DatabaseCredentials{
				Driver:   normalizeDriver(dbConfig.Driver),
				Host:     dbConfig.Host,
				Port:     dbConfig.Port,
				Name:     dbConfig.Name,
				User:     dbConfig.User,
				Password: dbConfig.Password,
			}, nil
		}
		return nil, fmt.Errorf("credential extraction failed: %w", err)
	}

	return creds, nil
}

func extractViaPhp(client *ssh.Client, deployPath, source string) (*DatabaseCredentials, error) {
	// Read embedded PHP script
	scriptContent, err := credentialsScript.ReadFile("credentials.php")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded credentials script: %w", err)
	}

	// Create local temp file
	// #nosec G404 -- Random suffix for temp file name, not security-sensitive
	remotePath := fmt.Sprintf("/tmp/.shippy-credentials-%d.php", rand.Int63())

	tmpFile, err := os.CreateTemp("", "shippy-credentials-*.php")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	localPath := tmpFile.Name()
	defer os.Remove(localPath)

	if _, err := tmpFile.Write(scriptContent); err != nil {
		_ = tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Upload to server
	if err := client.UploadFile(localPath, remotePath, 0o644); err != nil {
		return nil, fmt.Errorf("failed to upload credentials script: %w", err)
	}

	// Ensure cleanup
	defer func() {
		_, _ = client.RunCommand(fmt.Sprintf("rm -f %s", remotePath))
	}()

	// Execute PHP script
	sharedPath := filepath.Join(deployPath, "shared")
	currentPath := filepath.Join(deployPath, "current")
	output, err := client.RunCommand(fmt.Sprintf("php %s %s %s %s", remotePath, sharedPath, currentPath, source))
	if err != nil {
		return nil, fmt.Errorf("PHP credential extraction failed: %w\nOutput: %s", err, output)
	}

	// Parse JSON output
	output = strings.TrimSpace(output)
	var creds DatabaseCredentials
	if err := json.Unmarshal([]byte(output), &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials JSON: %w\nRaw output: %s", err, output)
	}

	if creds.Driver == "" || creds.Name == "" {
		return nil, fmt.Errorf("credential extraction returned incomplete data: driver=%q, name=%q", creds.Driver, creds.Name)
	}

	creds.Driver = normalizeDriver(creds.Driver)
	return &creds, nil
}

func normalizeDriver(driver string) string {
	switch strings.ToLower(driver) {
	case "mysqli", "pdo_mysql", "mysql", "mariadb":
		return "mysql"
	case "pgsql", "pdo_pgsql", "postgresql", "postgres":
		return "postgresql"
	case "sqlite3", "pdo_sqlite", "sqlite":
		return "sqlite"
	default:
		return strings.ToLower(driver)
	}
}
