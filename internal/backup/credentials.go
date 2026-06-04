package backup

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/ochorocho/shippy/internal/config"
	"github.com/ochorocho/shippy/internal/ssh"
)

//go:embed credentials.php
var credentialsScript embed.FS

// ExtractCredentials extracts database credentials from the remote server.
// It first checks for manual config, then TYPO3's configuration:show command,
// then falls back to a PHP script that reads the application's config files.
// The returned string describes which source the credentials came from.
func ExtractCredentials(client *ssh.Client, deployPath string, dbConfig *config.BackupDatabaseConfig) (*DatabaseCredentials, string, error) {
	// If manual credentials are fully specified, use them directly
	if dbConfig.HasManualCredentials() {
		return &DatabaseCredentials{
			Driver:   normalizeDriver(dbConfig.Driver),
			Host:     dbConfig.Host,
			Port:     dbConfig.Port,
			Name:     dbConfig.Name,
			User:     dbConfig.User,
			Password: dbConfig.Password,
		}, "manual config", nil
	}

	source := dbConfig.GetCredentialSource()

	// Prefer TYPO3 v14's `configuration:show` command, which returns the
	// authoritative active DB configuration. Fall back to PHP-script parsing
	// (older TYPO3, non-bootstrappable apps, Laravel/Symfony dotenv) on any failure.
	if source == "auto" || source == "typo3" {
		currentPath := filepath.Join(deployPath, "current")
		if creds, err := extractViaConsole(client, currentPath); err == nil {
			return creds, "typo3 configuration:show", nil
		}
	}

	// Extract credentials using PHP script on the remote server
	creds, err := extractViaPhp(client, deployPath, source)
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
			}, "partial manual config", nil
		}
		return nil, "", fmt.Errorf("credential extraction failed: %w", err)
	}

	return creds, "config files (.env / settings.php)", nil
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
		_, _ = client.RunCommand(fmt.Sprintf("rm -f %s", ssh.Quote(remotePath)))
	}()

	// Execute PHP script
	sharedPath := filepath.Join(deployPath, "shared")
	currentPath := filepath.Join(deployPath, "current")
	output, err := client.RunCommand(fmt.Sprintf("php %s %s %s %s",
		ssh.Quote(remotePath), ssh.Quote(sharedPath), ssh.Quote(currentPath), ssh.Quote(source)))
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

// extractViaConsole runs TYPO3 v14's `configuration:show` command in the active
// release directory to obtain the authoritative database connection settings.
func extractViaConsole(client *ssh.Client, currentPath string) (*DatabaseCredentials, error) {
	cmd := fmt.Sprintf("cd %s && ./vendor/bin/typo3 configuration:show --type=active --json DB/Connections/Default",
		ssh.Quote(currentPath))
	output, err := client.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("configuration:show failed: %w\nOutput: %s", err, output)
	}
	return parseConsoleCredentials(output)
}

// parseConsoleCredentials parses the JSON output of
// `typo3 configuration:show --json DB/Connections/Default` into credentials.
// TYPO3 uses the key "dbname" for the database name and reports drivers like
// "mysqli", which are normalized here.
func parseConsoleCredentials(output string) (*DatabaseCredentials, error) {
	// Isolate the JSON object so a stray warning/deprecation line on the same
	// stream does not break parsing.
	start := strings.IndexByte(output, '{')
	end := strings.LastIndexByte(output, '}')
	if start == -1 || end == -1 || end < start {
		return nil, fmt.Errorf("no JSON object found in configuration:show output: %q", output)
	}

	var parsed struct {
		Driver   string `json:"driver"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Name     string `json:"dbname"`
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(output[start:end+1]), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse configuration:show JSON: %w\nRaw output: %s", err, output)
	}

	if parsed.Driver == "" || parsed.Name == "" {
		return nil, fmt.Errorf("configuration:show returned incomplete data: driver=%q, dbname=%q", parsed.Driver, parsed.Name)
	}

	return &DatabaseCredentials{
		Driver:   normalizeDriver(parsed.Driver),
		Host:     parsed.Host,
		Port:     parsed.Port,
		Name:     parsed.Name,
		User:     parsed.User,
		Password: parsed.Password,
	}, nil
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
