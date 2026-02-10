package cmd

import (
	"github.com/spf13/cobra"
	"shippy/internal/composer"
	"shippy/internal/deploy"
	"shippy/internal/ui"
)

var (
	verbose bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [host]",
	Short: "Deploy to a target host",
	Long: `Deploy your TYPO3 project to a target host defined in .shippy.yaml.

The deployment process:
1. Scans files based on .gitignore and exclude patterns
2. Creates a new release directory with timestamp
3. Syncs files to the release directory
4. Creates symlinks for shared files/directories
5. Updates the "current" symlink to the new release
6. Executes post-deployment commands
7. Cleans up old releases (keeps last N releases)

Example:
  shippy deploy                  (interactive host selection)
  shippy deploy staging
  shippy deploy production`,
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

	out.Info("Loading configuration from: %s", cfgFile)
	cfg, err := loadConfigFile(out)
	if err != nil {
		return err
	}

	cfg.ApplyDefaults()

	if err := cfg.Validate(); err != nil {
		out.Error("Config validation failed: %v", err)
		return err
	}

	hostName, err := selectHostFromArgs(cfg, args, out)
	if err != nil {
		return err
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
