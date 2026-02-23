package backup

import (
	"fmt"
	"path/filepath"
	"time"

	"shippy/internal/config"
	"shippy/internal/ssh"
	"shippy/internal/ui"
)

// Backupper orchestrates the backup process
type Backupper struct {
	config   *config.Config
	host     *config.Host
	hostName string
	verbose  bool
}

// New creates a new Backupper
func New(cfg *config.Config, hostName string, verbose bool) (*Backupper, error) {
	host, err := cfg.GetHost(hostName)
	if err != nil {
		return nil, err
	}

	return &Backupper{
		config:   cfg,
		host:     host,
		hostName: hostName,
		verbose:  verbose,
	}, nil
}

// Backup performs the backup and returns the path to the created ZIP file
func (b *Backupper) Backup(skipDatabase, skipShared bool, outputOverride string) (string, error) {
	out := ui.New()

	out.Header("Shippy - Backup")
	out.Info("Target: %s (%s@%s:%s)", b.hostName, b.host.RemoteUser, b.host.Hostname, b.host.DeployPath)
	fmt.Printf("\n")

	// Step 1: Connect to server
	out.StepNumber(1, "Connecting to %s", b.host.Hostname)

	port := b.host.Port
	if port == 0 {
		port = 22
	}

	client, err := ssh.NewClientWithOptions(ssh.ClientOptions{
		Host:            b.host.Hostname,
		Port:            port,
		User:            b.host.RemoteUser,
		KeyPath:         b.host.SSHKey,
		SSHOptions:      b.host.SSHOptions,
		SSHMultiplexing: b.host.SSHMultiplexing,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := client.Connect(); err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	out.Success("Connected to %s@%s", b.host.RemoteUser, b.host.Hostname)

	// Determine output path
	outputDir := b.config.GetBackupOutput(b.host)
	if outputOverride != "" {
		outputDir = outputOverride
	}

	timestamp := time.Now().Format("20060102T150405")
	zipFileName := fmt.Sprintf("backup-%s-%s.zip", b.hostName, timestamp)
	zipPath := filepath.Join(outputDir, zipFileName)

	// Create ZIP builder
	zb, err := NewZipBuilder(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup archive: %w", err)
	}
	defer zb.Close()

	stepNum := 2

	// Step 2: Extract credentials and dump database
	if !skipDatabase {
		dbConfig := b.config.GetBackupDatabase(b.host)
		if dbConfig != nil {
			out.StepNumber(stepNum, "Extracting database credentials")
			stepNum++

			creds, err := ExtractCredentials(client, b.host.DeployPath, dbConfig)
			if err != nil {
				return "", fmt.Errorf("failed to extract database credentials: %w", err)
			}

			out.Success("Database: %s (%s@%s:%d/%s)", creds.Driver, creds.User, creds.Host, creds.Port, creds.Name)

			out.StepNumber(stepNum, "Dumping database")
			stepNum++

			dumpWriter, err := zb.CreateEntry("database/dump.sql")
			if err != nil {
				return "", fmt.Errorf("failed to create dump entry in archive: %w", err)
			}

			dumper, err := NewDumper(client, creds)
			if err != nil {
				return "", fmt.Errorf("failed to create database dumper: %w", err)
			}
			defer dumper.Close()

			if err := dumper.Dump(dumpWriter, dbConfig.ExcludeTables, dbConfig.Options); err != nil {
				return "", fmt.Errorf("database dump failed: %w", err)
			}

			out.Success("Database dumped successfully")
		} else {
			out.Info("No database configuration found, skipping database backup")
		}
	}

	// Step N: Download backup files
	if !skipShared {
		backupFiles := b.config.GetBackupFiles(b.host)
		if len(backupFiles) > 0 {
			out.StepNumber(stepNum, "Downloading files")
			stepNum++

			if err := DownloadShared(client, b.host.DeployPath, backupFiles, zb, b.verbose); err != nil {
				return "", fmt.Errorf("failed to download files: %w", err)
			}

			out.Success("Files downloaded")
		} else {
			out.Info("No backup files configured, skipping")
		}
	}

	// Finalize ZIP
	out.StepNumber(stepNum, "Creating backup archive")

	if err := zb.Close(); err != nil {
		return "", fmt.Errorf("failed to finalize backup archive: %w", err)
	}

	size, err := zb.Size()
	if err == nil {
		out.Success("Backup created: %s (%.2f MB)", zipPath, float64(size)/(1024*1024))
	} else {
		out.Success("Backup created: %s", zipPath)
	}

	out.HeaderGreen("Backup completed successfully!")

	return zipPath, nil
}
