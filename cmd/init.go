package cmd

import (
	"fmt"
	"os"

	"tinnie/internal/ui"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Tinnie configuration file",
	Long: `Initialize a new .tinnie.yaml configuration file with sensible TYPO3 defaults.

This command will:
- Check if composer.json exists
- Read project name from composer.json
- Generate a minimal .tinnie.yaml with TYPO3-optimized settings
- Prompt before overwriting existing configuration

Example:
  tinnie init`,
	RunE: runInit,
}

var (
	force bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing .tinnie.yaml file")
}

func runInit(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Header("Tinnie - Initialize Configuration")

	// Check if .tinnie.yaml already exists
	if _, err := os.Stat(cfgFile); err == nil && !force {
		out.Info("⚠ Configuration file already exists: %s", cfgFile)
		out.EmptyLine()
		out.Println("Use --force to overwrite the existing file:")
		out.Println("  tinnie init --force")
		out.EmptyLine()
		return nil
	}

	out.Step("Creating %s", cfgFile)

	configContent := generateMinimalConfig()

	// #nosec G306 -- Config file permissions 0644 are intentional for user readability
	if err := os.WriteFile(cfgFile, []byte(configContent), 0644); err != nil {
		out.Error("Failed to write config file: %v", err)
		return err
	}

	out.Success("Created %s", cfgFile)
	out.EmptyLine()

	out.HeaderGreen("Configuration initialized successfully!")

	out.Println("Next steps:")
	out.EmptyLine()
	out.Println("  1. Edit %s with your server details", cfgFile)
	out.Println("  2. Validate your configuration:")
	out.Println("     tinnie config validate")
	out.Println("  3. Deploy to your server:")
	out.Println("     tinnie deploy production")
	out.EmptyLine()

	return nil
}

func generateMinimalConfig() string {
	return fmt.Sprintf(`# Tinnie - TYPO3 Deployment Configuration
# Generated with: tinnie init

# Optional: Path to composer.json (default: composer.json in current directory)
# composer: composer.json

# Optional: Local directory to deploy (default: .)
# rsync_src: ./

# Optional: Number of releases to keep (default: 5)
# keep_releases: 5

# Optional: Shared files/directories (persistent across releases)
# These are symlinked from shared/ to each release
# shared:
#   - .env
#   - var/log/
#   - var/session/
#   - public/fileadmin/
#   - public/uploads/

# Optional: Files/patterns to exclude from deployment
# exclude:
#   - "*.log"
#   - ".env.example"

# Optional: Deployment locking (default: enabled, 15 min timeout)
# lock_enabled: true
# lock_timeout: 15

hosts:
  production:
    # Required: SSH connection details
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}

    # Optional: SSH port (default: 22)
    # port: 22

    # Optional: SSH key (auto-detects from ~/.ssh/ if not specified)
    # Always use the private key, not the .pub file
    # ssh_key: ~/.ssh/id_ed25519

    # Optional: Advanced SSH options (uncomment and adjust as needed)
    # For all available options, see: man ssh_config
    # ssh_options:
    #   ConnectTimeout: "10"
    #   ServerAliveInterval: "60"
    #   Compression: "yes"

    # Optional: SSH multiplexing - reuse one connection for all operations (default: false)
    # ssh_multiplexing: false

    # Optional: Per-host overrides for global settings
    # keep_releases: 3
    # rsync_src: ./dist

# Optional: Commands to run after deployment (executed in release directory)
# If omitted, default TYPO3 commands will be used:
#   - extension:setup
#   - upgrade:run
#   - language:update
#   - cache:flush
#   - cache:warmup
#
# Uncomment to customize:
# commands:
#   - name: Clear TYPO3 cache
#     run: ./vendor/bin/typo3 cache:flush
#
#   - name: Run extension setup
#     run: ./vendor/bin/typo3 extension:setup
#
#   - name: Database migrations
#     run: ./vendor/bin/typo3 upgrade:run
#
#   - name: Warmup caches
#     run: ./vendor/bin/typo3 cache:warmup
`)
}
