package cmd

import (
	"sort"

	"github.com/spf13/cobra"
	"tinnie/internal/composer"
	"tinnie/internal/config"
	"tinnie/internal/deploy"
	"tinnie/internal/ui"
)

var (
	verbose bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [host]",
	Short: "Deploy to a target host",
	Long: `Deploy your TYPO3 project to a target host defined in .tinnie.yaml.

The deployment process:
1. Scans files based on .gitignore and exclude patterns
2. Creates a new release directory with timestamp
3. Syncs files to the release directory
4. Creates symlinks for shared files/directories
5. Updates the "current" symlink to the new release
6. Executes post-deployment commands
7. Cleans up old releases (keeps last N releases)

Example:
  tinnie deploy                  (interactive host selection)
  tinnie deploy staging
  tinnie deploy production`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runDeploy,
	SilenceUsage: true, // Don't show usage on errors
}

func init() {
	rootCmd.AddCommand(deployCmd)
	deployCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed output for each file")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	out := ui.New()

	// Load config
	out.Info("Loading configuration from: %s", cfgFile)
	cfg, err := config.Load(cfgFile)
	if err != nil {
		out.Error("Failed to load config: %v", err)
		return err
	}

	// Apply defaults before validation
	cfg.ApplyDefaults()

	// Validate config
	if err := cfg.Validate(); err != nil {
		out.Error("Config validation failed: %v", err)
		return err
	}

	// Determine host: from args or interactive selection
	var hostName string
	if len(args) > 0 {
		hostName = args[0]
	} else {
		// Get available hosts sorted alphabetically
		hosts := make([]string, 0, len(cfg.Hosts))
		for name := range cfg.Hosts {
			hosts = append(hosts, name)
		}
		sort.Strings(hosts)

		// Prompt for selection
		selected, err := out.SelectHost(hosts)
		if err != nil {
			out.Error("Host selection failed: %v", err)
			return err
		}
		hostName = selected
	}

	// Load composer.json using configured path
	composerPath := cfg.GetComposerPath()
	comp, err := composer.Parse(composerPath)
	if err != nil {
		out.Error("Failed to load composer.json from %s: %v", composerPath, err)
		return err
	}

	// Process templates
	if err := cfg.ProcessTemplates(comp); err != nil {
		out.Error("Template processing failed: %v", err)
		return err
	}

	// Create deployer
	deployer, err := deploy.New(cfg, comp, hostName, verbose)
	if err != nil {
		out.Error("Failed to create deployer: %v", err)
		return err
	}

	// Deploy!
	if err := deployer.Deploy(); err != nil {
		out.Error("Deployment failed: %v", err)
		return err
	}

	return nil
}
