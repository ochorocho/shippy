package deploy

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/ochorocho/shippy/internal/composer"
	"github.com/ochorocho/shippy/internal/config"
	"github.com/ochorocho/shippy/internal/lock"
	"github.com/ochorocho/shippy/internal/rsync"
	"github.com/ochorocho/shippy/internal/ssh"
	"github.com/ochorocho/shippy/internal/ui"
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

	includePatterns := d.config.GetInclude(d.host)
	if len(includePatterns) == 0 {
		out.Warning("No include patterns configured - deployment is deny-by-default, so nothing will be synced.")
		out.Info("  Add an 'include:' allowlist (e.g. public/, vendor/, config/, composer.json) to your .shippy.yaml.")
	}

	scanOpts := rsync.SyncOptions{
		SourceDir:       d.config.GetRsyncSrc(d.host),
		ExcludePatterns: d.getExcludePatterns(),
		IncludePatterns: includePatterns,
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

	// Acquire deployment lock (if enabled)
	if d.config.IsLockEnabled(d.host) {
		timeout := time.Duration(d.config.GetLockTimeout(d.host)) * time.Minute
		locker := lock.NewLocker(client, d.host.DeployPath, timeout)

		if err := locker.Acquire(fmt.Sprintf("Deploying to %s", d.hostName)); err != nil {
			return err
		}
		defer locker.Release()

		out.Info("  Deployment lock acquired (expires in %d minutes)", d.config.GetLockTimeout(d.host))
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

	// Write release metadata (best-effort, non-critical)
	if err := WriteReleaseMetadata(client, releasePath); err != nil {
		out.Info("  Could not write release metadata: %v", err)
	} else {
		out.Success("Release metadata written")
	}

	// Step 5: Create shared symlinks
	sharedItems := d.config.GetShared(d.host)
	if len(sharedItems) > 0 {
		out.StepNumber(5, "Creating shared symlinks")

		if err := releaseMgr.CreateSharedSymlinks(releasePath, sharedItems); err != nil {
			return fmt.Errorf("failed to create shared symlinks: %w", err)
		}

		out.Success("Created %d shared symlinks", len(sharedItems))
	}

	// Step 6: Execute commands in the new release
	if len(d.config.Commands) > 0 {
		out.StepNumber(6, "Executing commands")

		executor := ssh.NewExecutor(client)

		// Convert config commands to SSH commands, skipping any not enabled
		// for this host via only/except.
		var commands []ssh.Command
		for _, cmd := range d.config.Commands {
			if !cmd.AppliesToHost(d.hostName) {
				out.Info("  Skipping %s (not enabled for %s)", cmd.Name, d.hostName)
				continue
			}
			commands = append(commands, ssh.Command{
				Name:    cmd.Name,
				Run:     cmd.Run,
				Context: d.config.GetCommandContext(d.host, cmd),
			})
		}

		// Execute commands in the specific release directory (NOT the current symlink)
		// This ensures commands run against the new release before it goes live
		if len(commands) > 0 {
			if err := executor.Execute(commands, releasePath); err != nil {
				return fmt.Errorf("command execution failed: %w", err)
			}
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

	keepReleases := d.config.GetKeepReleases(d.host)

	if err := releaseMgr.CleanupOldReleases(keepReleases); err != nil {
		return fmt.Errorf("failed to cleanup old releases: %w", err)
	}

	out.Success("Kept last %d releases", keepReleases)

	// Success!
	out.HeaderGreen("Deployment completed successfully!")

	return nil
}

// DryRun previews which files and commands would be deployed without
// connecting to the remote host. The file scan runs entirely locally, so this
// works even when the target is unreachable.
func (d *Deployer) DryRun() error {
	out := ui.New()

	out.Header("Shippy - Dry Run (no changes will be made)")
	out.Info("Target: %s (%s@%s:%s)", d.hostName, d.host.RemoteUser, d.host.Hostname, d.host.DeployPath)
	fmt.Printf("\n")

	// Scan files (local only, no SSH connection required)
	out.StepNumber(1, "Scanning files")

	includePatterns := d.config.GetInclude(d.host)
	if len(includePatterns) == 0 {
		out.Warning("No include patterns configured - deployment is deny-by-default, so nothing will be synced.")
		out.Info("  Add an 'include:' allowlist (e.g. public/, vendor/, config/, composer.json) to your .shippy.yaml.")
	}

	scanOpts := rsync.SyncOptions{
		SourceDir:       d.config.GetRsyncSrc(d.host),
		ExcludePatterns: d.getExcludePatterns(),
		IncludePatterns: includePatterns,
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
	d.printFileTree(out, files)

	// Shared symlinks that would be created
	sharedItems := d.config.GetShared(d.host)
	if len(sharedItems) > 0 {
		out.StepNumber(2, "Shared symlinks (%d)", len(sharedItems))
		for _, item := range sharedItems {
			out.Println("  %s", item)
		}
	}

	// Commands that would be executed in the new release
	if len(d.config.Commands) > 0 {
		out.StepNumber(3, "Commands to execute (%d)", len(d.config.Commands))
		for _, cmd := range d.config.Commands {
			out.Println("  %s", cmd.Name)
			out.Info("    $ %s", cmd.Run)
		}
	}

	out.HeaderGreen("Dry run completed - nothing was deployed")

	return nil
}

// printFileTree groups scanned files by directory. By default only directories
// with their file count are shown; with verbose the individual files are listed.
func (d *Deployer) printFileTree(out *ui.Output, files []rsync.FileInfo) {
	groups := make(map[string][]string)
	for _, f := range files {
		dir := path.Dir(f.RelPath)
		if dir == "." {
			dir = "./"
		}
		groups[dir] = append(groups[dir], f.RelPath)
	}

	dirs := make([]string, 0, len(groups))
	for dir := range groups {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		entries := groups[dir]
		out.Println("  %s (%d files)", dir, len(entries))
		if d.verbose {
			sort.Strings(entries)
			for _, rel := range entries {
				out.Info("    %s", rel)
			}
		}
	}
}

// getExcludePatterns returns the carve-out patterns: the built-in junk list plus
// any user-defined excludes. These always win over the include allowlist.
func (d *Deployer) getExcludePatterns() []string {
	// Start from a fresh slice so we never mutate the package-level
	// defaultExcludePatterns backing array across calls.
	excludes := make([]string, 0, len(defaultExcludePatterns)+len(d.config.GetExclude(d.host)))
	excludes = append(excludes, defaultExcludePatterns...)
	excludes = append(excludes, d.config.GetExclude(d.host)...)
	return excludes
}
