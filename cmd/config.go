package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"shippy/internal/composer"
	"shippy/internal/config"
	"shippy/internal/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	RunE:  runValidate,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	out := ui.New()

	fmt.Println("Validating configuration...")
	fmt.Println()

	// Load config
	out.Step("Loading config from: %s", cfgFile)
	cfg, err := config.Load(cfgFile)
	if err != nil {
		out.Error("Failed to load config: %v", err)
		return err
	}
	out.Success("Config file loaded successfully")

	// Validate structure
	out.Step("Validating config structure")
	if err := cfg.Validate(); err != nil {
		out.Error("Validation failed: %v", err)
		return err
	}
	out.Success("Config structure is valid")

	// Parse composer.json
	out.Step("Loading composer.json")
	comp, err := composer.Parse("composer.json")
	if err != nil {
		out.Error("Failed to load composer.json: %v", err)
		return err
	}
	out.Success("composer.json loaded successfully")

	// Process templates
	out.Step("Processing template variables")
	if err := cfg.ProcessTemplates(comp); err != nil {
		out.Error("Template processing failed: %v", err)
		return err
	}
	out.Success("Template variables processed successfully")

	// Re-validate after template processing
	out.Step("Validating processed configuration")
	if err := cfg.Validate(); err != nil {
		out.Error("Validation failed after template processing: %v", err)
		return err
	}
	out.Success("Processed configuration is valid")

	fmt.Println()
	out.Success("Configuration is valid!")
	fmt.Println()

	// Print host summary
	fmt.Printf("Configured hosts (%d):\n", len(cfg.Hosts))
	for name, host := range cfg.Hosts {
		fmt.Printf("  • %s\n", name)
		if host.Port != 0 && host.Port != 22 {
			fmt.Printf("    Hostname: %s:%d\n", host.Hostname, host.Port)
		} else {
			fmt.Printf("    Hostname: %s\n", host.Hostname)
		}
		fmt.Printf("    User: %s\n", host.RemoteUser)
		fmt.Printf("    Deploy path: %s\n", host.DeployPath)
		fmt.Printf("    Source: %s\n", host.RsyncSrc)
		if host.SSHKey != "" {
			fmt.Printf("    SSH key: %s\n", host.SSHKey)
		}
		if len(host.SSHOptions) > 0 {
			fmt.Printf("    SSH options: %d configured\n", len(host.SSHOptions))
		}
		if len(host.Exclude) > 0 {
			fmt.Printf("    Excludes: %d patterns\n", len(host.Exclude))
		}
		if len(host.Include) > 0 {
			fmt.Printf("    Includes: %d patterns\n", len(host.Include))
		}
		fmt.Println()
	}

	if len(cfg.Commands) > 0 {
		fmt.Printf("Configured commands (%d):\n", len(cfg.Commands))
		for i, cmd := range cfg.Commands {
			fmt.Printf("  %d. %s\n", i+1, cmd.Name)
			fmt.Printf("     %s\n", cmd.Run)
		}
	}

	return nil
}
