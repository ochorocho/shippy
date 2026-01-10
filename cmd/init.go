package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"shippy/internal/composer"
	"shippy/internal/ui"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Shippy configuration file",
	Long: `Initialize a new .shippy.yaml configuration file with sensible TYPO3 defaults.

This command will:
- Check if composer.json exists
- Read project name from composer.json
- Generate a minimal .shippy.yaml with TYPO3-optimized settings
- Prompt before overwriting existing configuration

Example:
  shippy init`,
	RunE: runInit,
}

var (
	force bool
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing .shippy.yaml file")
}

func runInit(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Header("Shippy - Initialize Configuration")

	// Check if .shippy.yaml already exists
	if _, err := os.Stat(cfgFile); err == nil && !force {
		out.Info("⚠ Configuration file already exists: %s", cfgFile)
		fmt.Println()
		fmt.Println("Use --force to overwrite the existing file:")
		fmt.Printf("  shippy init --force\n")
		fmt.Println()
		return nil
	}

	// Load composer.json
	out.Step("Loading composer.json")
	comp, err := composer.Parse("composer.json")
	if err != nil {
		out.Error("Failed to load composer.json: %v", err)
		fmt.Println()
		out.Info("Make sure you run this command in a directory with a composer.json file.")
		return err
	}
	out.Success("composer.json loaded")
	fmt.Println()

	// Get project name
	projectName, err := comp.GetString("name")
	if err != nil {
		out.Info("⚠ Could not read project name from composer.json, using 'myproject'")
		projectName = "myproject"
	} else {
		out.Success("Project: %s", projectName)
	}
	fmt.Println()

	// Generate minimal config
	out.Step("Creating %s", cfgFile)

	configContent := generateMinimalConfig(projectName)

	if err := os.WriteFile(cfgFile, []byte(configContent), 0644); err != nil {
		out.Error("Failed to write config file: %v", err)
		return err
	}

	out.Success("Created %s", cfgFile)
	fmt.Println()

	// Success message
	out.HeaderGreen("Configuration initialized successfully!")

	fmt.Println("Next steps:")
	fmt.Println()
	fmt.Printf("  1. Edit %s with your server details\n", cfgFile)
	fmt.Println("  2. Validate your configuration:")
	fmt.Println("     shippy config validate")
	fmt.Println("  3. Deploy to your server:")
	fmt.Println("     shippy deploy production")
	fmt.Println()

	return nil
}

func generateMinimalConfig(projectName string) string {
	return fmt.Sprintf(`# Shippy - TYPO3 Deployment Configuration
# Generated with: shippy init

hosts:
  production:
    # SSH connection details
    hostname: www.example.com
    port: 22  # SSH port (default: 22, can be omitted)
    remote_user: deploy

    # SSH key (optional - auto-detects from ~/.ssh/ if not specified)
    # Always use the private key, not the .pub file
    # ssh_key: ~/.ssh/id_ed25519

    # Optional: Advanced SSH options (uncomment and adjust as needed)
    # For all available options, see: man ssh_config
    # ssh_options:
    #   ConnectTimeout: "10"
    #   ServerAliveInterval: "60"
    #   Compression: "yes"

    # SSH multiplexing: reuse one connection for all operations (faster)
    # Default: false (disabled for compatibility)
    ssh_multiplexing: false

    # Deployment path on the server
    # Uses {{name}} template variable from composer.json
    deploy_path: /var/www/{{name}}

    # Local directory to deploy
    rsync_src: ./

    # Number of releases to keep (default: 5)
    keep_releases: 5

    # Shared files/directories (persistent across releases)
    # These are symlinked from shared/ to each release
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

# Commands to run after deployment (executed in release directory)
commands:
  - name: Clear TYPO3 cache
    run: ./vendor/bin/typo3 cache:flush

  - name: Run extension setup
    run: ./vendor/bin/typo3 extension:setup

  - name: Database migrations
    run: ./vendor/bin/typo3 upgrade:run

  - name: Warmup caches
    run: ./vendor/bin/typo3 cache:warmup
`)
}
