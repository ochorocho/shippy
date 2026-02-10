package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"shippy/internal/deploy"
	"shippy/internal/ssh"
	"shippy/internal/ui"
)

var (
	rollbackRelease string
	rollbackOffset  int
	rollbackList    bool
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback [host]",
	Short: "Rollback to a previous release",
	Long: `Rollback to a previous release by switching the "current" symlink.

Shows a list of available releases with deployment date/time, git commit hash,
and git tag. Select a release to rollback to.

Use --release to switch to a specific release by name, or --offset to move
relative to the current release (negative = older, positive = newer).

Example:
  shippy rollback production                    (interactive release selection)
  shippy rollback production -l                 (list available releases)
  shippy rollback production -r 20260109120000  (specific release)
  shippy rollback production -n -1              (one version back)
  shippy rollback production -n +1              (one version forward)
  shippy rollback production -n -2              (two versions back)`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runRollback,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(rollbackCmd)
	rollbackCmd.Flags().StringVarP(&rollbackRelease, "release", "r", "", "switch to a specific release by name")
	rollbackCmd.Flags().IntVarP(&rollbackOffset, "offset", "n", 0, "relative offset from current release (negative = older, positive = newer)")
	rollbackCmd.Flags().BoolVarP(&rollbackList, "list", "l", false, "list available releases and exit")
}

func runRollback(cmd *cobra.Command, args []string) error {
	out := ui.New()

	// Validate flag combination
	if rollbackList && (rollbackRelease != "" || rollbackOffset != 0) {
		out.Error("cannot use --list with --release or --offset")
		return fmt.Errorf("cannot use --list with --release or --offset")
	}
	if rollbackRelease != "" && rollbackOffset != 0 {
		out.Error("cannot use --release and --offset together")
		return fmt.Errorf("cannot use --release and --offset together")
	}

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

	// List mode: print releases and exit
	if rollbackList {
		printReleaseList(items, out)
		return nil
	}

	var selected *ui.ReleaseItem

	switch {
	case rollbackRelease != "":
		// Find specific release by name
		selected, err = findReleaseByName(items, rollbackRelease, out)
		if err != nil {
			return err
		}

	case rollbackOffset != 0:
		// Find release by relative offset from current
		selected, err = findReleaseByOffset(items, rollbackOffset, out)
		if err != nil {
			return err
		}

	default:
		// Check if there are any non-current releases to rollback to
		hasRollbackTarget := false
		for _, item := range items {
			if !item.IsCurrent {
				hasRollbackTarget = true
				break
			}
		}
		if !hasRollbackTarget {
			out.Info("Only one release exists - nothing to rollback to")
			return fmt.Errorf("no previous releases available for rollback")
		}

		// Show interactive release selector
		selected, err = out.SelectRelease(items)
		if err != nil {
			out.Error("Release selection failed: %v", err)
			return err
		}
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

// findReleaseByName finds a release by its exact name
func findReleaseByName(items []ui.ReleaseItem, name string, out *ui.Output) (*ui.ReleaseItem, error) {
	for i, item := range items {
		if item.Name == name {
			if item.IsCurrent {
				out.Error("Release %s is already the current release", name)
				return nil, fmt.Errorf("release %s is already active", name)
			}
			return &items[i], nil
		}
	}
	out.Error("Can't rollback to release %s because it does not exist", name)
	return nil, fmt.Errorf("can't rollback to release %s because it does not exist", name)
}

// findReleaseByOffset finds a release by relative offset from the current release.
// Releases are sorted newest-first. Negative offset = older, positive = newer.
func findReleaseByOffset(items []ui.ReleaseItem, offset int, out *ui.Output) (*ui.ReleaseItem, error) {
	// Find current release index
	currentIdx := -1
	for i, item := range items {
		if item.IsCurrent {
			currentIdx = i
			break
		}
	}
	if currentIdx == -1 {
		return nil, fmt.Errorf("could not determine current release")
	}

	// Calculate target index
	// Releases are newest-first, so offset -1 (older) means index+1, offset +1 (newer) means index-1
	targetIdx := currentIdx - offset

	if targetIdx < 0 || targetIdx >= len(items) {
		direction := "older"
		if offset > 0 {
			direction = "newer"
		}
		out.Error("Can't rollback to release at offset %+d because it does not exist (no %s release available)", offset, direction)
		return nil, fmt.Errorf("can't rollback to release at offset %+d because it does not exist (no %s release available)", offset, direction)
	}

	if targetIdx == currentIdx {
		return nil, fmt.Errorf("offset 0 targets the current release — nothing to do")
	}

	return &items[targetIdx], nil
}

// printReleaseList prints a formatted list of all available releases
func printReleaseList(items []ui.ReleaseItem, out *ui.Output) {
	out.EmptyLine()
	out.Info("Available releases:")
	out.EmptyLine()
	for _, item := range items {
		line := "  " + item.Name
		if item.DateTime != "" {
			line += "  " + item.DateTime
		}
		if item.GitCommit != "" {
			line += "  " + item.GitCommit
		}
		if item.GitTag != "" {
			line += "  " + item.GitTag
		}
		if item.IsCurrent {
			line += "  (current)"
		}
		out.Println("%s", line)
	}
	out.EmptyLine()
	out.Println("Total: %d releases", len(items))
}
