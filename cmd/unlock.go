package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"tinnie/internal/config"
	"tinnie/internal/lock"
	"tinnie/internal/ssh"
	"tinnie/internal/ui"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock [host]",
	Short: "Force unlock a deployment target",
	Long: `Removes the deployment lock from the specified host.

This command should only be used when a deployment has failed and left a stale lock,
or when you need to force-unlock a deployment that is stuck.

WARNING: Using this command while a deployment is in progress may cause issues.

Example:
  tinnie unlock                  (interactive host selection)
  tinnie unlock production
  tinnie unlock staging`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runUnlock,
	SilenceUsage: true, // Don't show usage on errors
}

func init() {
	rootCmd.AddCommand(unlockCmd)
}

func runUnlock(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Info("Loading configuration from: %s", cfgFile)
	cfg, err := loadConfigFile(out)
	if err != nil {
		return err
	}

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

	// Check lock status
	locker := lock.NewLocker(client, host.DeployPath, time.Duration(config.DefaultLockTimeoutMinutes)*time.Minute)
	locked, info, err := locker.IsLocked()
	if err != nil {
		out.Error("Failed to check lock status: %v", err)
		return err
	}

	if !locked {
		out.Info("No active deployment lock found")
		return nil
	}

	out.EmptyLine()
	out.Info("Current lock information:")
	out.Println("  Locked by: %s", info.LockedBy)
	out.Println("  Locked at: %s", info.LockedAt.Format("2006-01-02 15:04:05"))
	out.Println("  Expires:   %s", info.ExpiresAt.Format("2006-01-02 15:04:05"))
	out.Println("  Message:   %s", info.Message)
	out.EmptyLine()

	// Force unlock
	if err := locker.ForceUnlock(); err != nil {
		out.Error("Failed to unlock: %v", err)
		return err
	}

	out.Success("Deployment lock removed successfully")
	return nil
}
