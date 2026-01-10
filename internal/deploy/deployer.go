package deploy

import (
	"fmt"
	"path/filepath"

	"shippy/internal/composer"
	"shippy/internal/config"
	"shippy/internal/rsync"
	"shippy/internal/ssh"
	"shippy/internal/ui"
)

// defaultExcludePatterns contains patterns that should always be excluded from deployment
var defaultExcludePatterns = []string{
	".git/",
	".gitignore",
	".shippy.yaml",
	".shippy.yaml.example",
	"node_modules/",
	".env.local",
	".env.*.local",
	"var/cache/",
	"var/log/",
	"var/transient/",
	".DS_Store",
	"Thumbs.db",
}

// Deployer orchestrates the deployment process
type Deployer struct {
	config   *config.Config
	composer *composer.Composer
	host     *config.Host
	hostName string
	verbose  bool
}

// New creates a new deployer
func New(cfg *config.Config, comp *composer.Composer, hostName string, verbose bool) (*Deployer, error) {
	host, err := cfg.GetHost(hostName)
	if err != nil {
		return nil, err
	}

	return &Deployer{
		config:   cfg,
		composer: comp,
		host:     host,
		hostName: hostName,
		verbose:  verbose,
	}, nil
}

// Deploy performs the deployment
func (d *Deployer) Deploy() error {
	out := ui.New()

	out.Header("Shippy - TYPO3 Deployment Tool")
	out.Info("Target: %s (%s@%s:%s)", d.hostName, d.host.RemoteUser, d.host.Hostname, d.host.DeployPath)
	fmt.Printf("\n")

	// Step 1: Scan files
	out.StepNumber(1, "Scanning files")

	scanOpts := rsync.SyncOptions{
		SourceDir:       d.host.RsyncSrc,
		ExcludePatterns: d.getExcludePatterns(),
		IncludePatterns: d.host.Include,
		UseGitignore:    true,
	}

	scanner, err := rsync.NewScanner(scanOpts)
	if err != nil {
		return fmt.Errorf("failed to create scanner: %w", err)
	}

	files, err := scanner.Scan()
	if err != nil {
		return fmt.Errorf("failed to scan files: %w", err)
	}

	out.Success("Found %d files to sync", len(files))

	// Step 2: Connect to server
	out.StepNumber(2, "Connecting to %s", d.host.Hostname)

	// Build SSH client options
	port := d.host.Port
	if port == 0 {
		port = 22 // Default SSH port
	}

	client, err := ssh.NewClientWithOptions(ssh.ClientOptions{
		Host:            d.host.Hostname,
		Port:            port,
		User:            d.host.RemoteUser,
		KeyPath:         d.host.SSHKey,
		SSHOptions:      d.host.SSHOptions,
		SSHMultiplexing: d.host.SSHMultiplexing,
	})
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	if port != 22 {
		out.Success("Connected to %s@%s:%d", d.host.RemoteUser, d.host.Hostname, port)
	} else {
		out.Success("Connected to %s@%s", d.host.RemoteUser, d.host.Hostname)
	}

	if d.host.SSHMultiplexing {
		out.Info("  SSH multiplexing enabled - reusing connection for all operations")
	}

	// Step 3: Create new release
	out.StepNumber(3, "Creating new release")

	releaseMgr := NewReleaseManager(client, d.host.DeployPath)
	releasePath, err := releaseMgr.CreateRelease()
	if err != nil {
		return fmt.Errorf("failed to create release: %w", err)
	}

	out.Success("Created release: %s", filepath.Base(releasePath))

	// Step 4: Sync files to release
	out.StepNumber(4, "Syncing files to release")

	syncer := rsync.NewSyncer(client, releasePath, d.verbose, d.host.DeployPath)
	if err := syncer.Sync(files); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	// Step 5: Create shared symlinks
	if len(d.host.Shared) > 0 {
		out.StepNumber(5, "Creating shared symlinks")

		if err := releaseMgr.CreateSharedSymlinks(releasePath, d.host.Shared); err != nil {
			return fmt.Errorf("failed to create shared symlinks: %w", err)
		}

		out.Success("Created %d shared symlinks", len(d.host.Shared))
	}

	// Step 6: Execute commands in the new release
	if len(d.config.Commands) > 0 {
		out.StepNumber(6, "Executing commands")

		executor := ssh.NewExecutor(client)

		// Convert config commands to SSH commands
		commands := make([]ssh.Command, len(d.config.Commands))
		for i, cmd := range d.config.Commands {
			commands[i] = ssh.Command{Name: cmd.Name, Run: cmd.Run}
		}

		// Execute commands in the specific release directory (NOT the current symlink)
		// This ensures commands run against the new release before it goes live
		if err := executor.Execute(commands, releasePath); err != nil {
			return fmt.Errorf("command execution failed: %w", err)
		}
	}

	// Step 7: Activate release (atomic switchover)
	out.StepNumber(7, "Activating release (atomic switchover)")

	if err := releaseMgr.UpdateCurrentSymlink(releasePath); err != nil {
		return fmt.Errorf("failed to update current symlink: %w", err)
	}

	out.Success("Release activated - site is now live!")

	// Step 8: Cleanup old releases
	out.StepNumber(8, "Cleaning up old releases")

	keepReleases := d.host.KeepReleases
	if keepReleases == 0 {
		keepReleases = 5 // Default
	}

	if err := releaseMgr.CleanupOldReleases(keepReleases); err != nil {
		return fmt.Errorf("failed to cleanup old releases: %w", err)
	}

	out.Success("Kept last %d releases", keepReleases)

	// Success!
	out.HeaderGreen("Deployment completed successfully!")

	return nil
}

// getExcludePatterns returns the combined list of exclude patterns
func (d *Deployer) getExcludePatterns() []string {
	// Combine default excludes with user-defined excludes
	return append(defaultExcludePatterns, d.host.Exclude...)
}
