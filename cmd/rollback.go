package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"shippy/internal/deploy"
	"shippy/internal/ssh"
	"shippy/internal/ui"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [host]",
	Short: "Rollback to a previous release",
	Long: `Rollback to a previous release by switching the "current" symlink.

Shows a list of available releases with deployment date/time, git commit hash,
and git tag. Select a release to rollback to.

Example:
  shippy rollback                  (interactive host selection)
  shippy rollback production`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runRollback,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
}

func runRollback(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Header("Shippy - Rollback")

	out.Info("Loading configuration from: %s", cfgFile)
	cfg, err := loadConfigFile(out)
	if err != nil {
		return err
	}

	cfg.ApplyDefaults()

	hostName, err := selectHostFromArgs(cfg, args, out)
	if err != nil {
		return err
	}

	host, err := cfg.GetHost(hostName)
	if err != nil {
		out.Error("Failed to get host config: %v", err)
		return err
	}

	// Connect to SSH
	out.Info("Connecting to %s@%s", host.RemoteUser, host.Hostname)

	port := host.Port
	if port == 0 {
		port = 22
	}

	client, err := ssh.NewClientWithOptions(ssh.ClientOptions{
		Host:            host.Hostname,
		Port:            port,
		User:            host.RemoteUser,
		KeyPath:         host.SSHKey,
		SSHOptions:      host.SSHOptions,
		SSHMultiplexing: host.SSHMultiplexing,
	})
	if err != nil {
		out.Error("Failed to create SSH client: %v", err)
		return err
	}

	if err := client.Connect(); err != nil {
		out.Error("Failed to connect: %v", err)
		return err
	}
	defer client.Close()

	out.Success("Connected to %s@%s", host.RemoteUser, host.Hostname)

	// List available releases
	releaseMgr := deploy.NewReleaseManager(client, host.DeployPath)
	releases, err := releaseMgr.ListReleases()
	if err != nil {
		out.Error("Failed to list releases: %v", err)
		return err
	}

	if len(releases) == 0 {
		out.Info("No releases found")
		return fmt.Errorf("no releases available for rollback")
	}

	// Check if there are any non-current releases to rollback to
	hasRollbackTarget := false
	for _, r := range releases {
		if !r.IsCurrent {
			hasRollbackTarget = true
			break
		}
	}
	if !hasRollbackTarget {
		out.Info("Only one release exists - nothing to rollback to")
		return fmt.Errorf("no previous releases available for rollback")
	}

	// Build UI items from release info
	items := make([]ui.ReleaseItem, len(releases))
	for i, r := range releases {
		item := ui.ReleaseItem{
			Name:      r.Name,
			Path:      r.Path,
			DateTime:  ui.FormatReleaseDateTime(r.Name),
			IsCurrent: r.IsCurrent,
		}
		if r.Metadata != nil {
			item.GitCommit = r.Metadata.GitCommit
			item.GitTag = r.Metadata.GitTag
		}
		items[i] = item
	}

	// Show interactive release selector
	selected, err := out.SelectRelease(items)
	if err != nil {
		out.Error("Release selection failed: %v", err)
		return err
	}

	out.EmptyLine()
	out.Info("Rolling back to release: %s", selected.Name)
	if selected.DateTime != "" {
		out.Println("  Deployed: %s", selected.DateTime)
	}
	if selected.GitCommit != "" {
		out.Println("  Commit:   %s", selected.GitCommit)
	}
	if selected.GitTag != "" {
		out.Println("  Tag:      %s", selected.GitTag)
	}
	out.EmptyLine()

	// Switch the current symlink
	if err := releaseMgr.UpdateCurrentSymlink(selected.Path); err != nil {
		out.Error("Failed to switch release: %v", err)
		return err
	}

	out.HeaderGreen(fmt.Sprintf("Rolled back to %s", selected.Name))

	return nil
}
