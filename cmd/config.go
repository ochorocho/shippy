package cmd

import (
	"fmt"
	"strings"

	"tinnie/internal/composer"
	"tinnie/internal/config"
	"tinnie/internal/ui"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
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

var (
	rawOutput bool
)

var showCmd = &cobra.Command{
	Use:   "show [host]",
	Short: "Show the complete resolved configuration",
	Long: `Display the complete configuration with all defaults applied and templates resolved.

If a host is specified, shows the effective configuration for that host
including global settings and per-host overrides.

By default, template variables are resolved. Use --raw to show unresolved templates.

Examples:
  tinnie config show              # Show complete config with resolved templates
  tinnie config show --raw        # Show raw config without resolving templates
  tinnie config show production   # Show config for production host`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(showCmd)
	showCmd.Flags().BoolVar(&rawOutput, "raw", false, "Show raw config without resolving template variables")
}

func runValidate(cmd *cobra.Command, args []string) error {
	out := ui.New()

	out.Println("Validating configuration...")
	out.EmptyLine()

	out.Step("Loading config from: %s", cfgFile)
	cfg, err := loadConfigFile(out)
	if err != nil {
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

	// Parse composer.json using configured path
	composerPath := cfg.GetComposerPath()
	out.Step("Loading composer.json from: %s", composerPath)
	comp, err := composer.Parse(composerPath)
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

	out.EmptyLine()
	out.Success("Configuration is valid!")
	out.EmptyLine()

	out.Println("Configured hosts (%d):", len(cfg.Hosts))
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
			fmt.Printf("    SSH options:\n")
			for key, value := range host.SSHOptions {
				fmt.Printf("      • %s: %s\n", key, value)
			}
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

func runShow(cmd *cobra.Command, args []string) error {
	out := ui.New()

	cfg, err := loadConfigFile(out)
	if err != nil {
		return err
	}

	cfg.ApplyDefaults()

	if !rawOutput {
		composerPath := cfg.GetComposerPath()
		comp, err := composer.Parse(composerPath)
		if err != nil {
			out.Error("Failed to load composer.json from %s: %v", composerPath, err)
			return err
		}

		if err := cfg.ProcessTemplates(comp); err != nil {
			out.Error("Template processing failed: %v", err)
			return err
		}
	}

	if len(args) > 0 {
		hostName := args[0]
		host, err := cfg.GetHost(hostName)
		if err != nil {
			out.Error("Host not found: %v", err)
			return err
		}

		return showHostConfig(cfg, host, hostName, out)
	}

	return showCompleteConfig(cfg, out)
}

func showCompleteConfig(cfg *config.Config, out *ui.Output) error {
	out.Header("Complete Configuration (with defaults applied)")
	out.EmptyLine()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	out.PrintYAML(string(data))
	return nil
}

func showHostConfig(cfg *config.Config, host *config.Host, hostName string, out *ui.Output) error {
	out.Header(fmt.Sprintf("Configuration for host: %s", hostName))
	out.EmptyLine()

	resolvedConfig := buildResolvedHostConfig(cfg, host, hostName)
	out.PrintYAML(resolvedConfig)
	return nil
}

func buildResolvedHostConfig(cfg *config.Config, host *config.Host, hostName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Resolved configuration for: %s\n", hostName))
	sb.WriteString("# (includes global defaults and per-host overrides)\n\n")

	// Global settings section
	sb.WriteString("# Global Settings\n")
	sb.WriteString(fmt.Sprintf("rsync_src: %s\n", cfg.GetRsyncSrc(host)))
	sb.WriteString(fmt.Sprintf("exclude: %s\n", formatStringSlice(cfg.GetExclude(host))))
	sb.WriteString(fmt.Sprintf("include: %s\n", formatStringSlice(cfg.GetInclude(host))))
	sb.WriteString(fmt.Sprintf("shared: %s\n", formatStringSlice(cfg.GetShared(host))))
	sb.WriteString(fmt.Sprintf("keep_releases: %d\n", cfg.GetKeepReleases(host)))
	sb.WriteString(fmt.Sprintf("lock_enabled: %t\n", cfg.IsLockEnabled(host)))
	sb.WriteString(fmt.Sprintf("lock_timeout: %d\n\n", cfg.GetLockTimeout(host)))

	// Host details
	sb.WriteString("# Host Configuration\n")
	sb.WriteString(fmt.Sprintf("hostname: %s\n", host.Hostname))
	if host.Port > 0 {
		sb.WriteString(fmt.Sprintf("port: %d\n", host.Port))
	}
	sb.WriteString(fmt.Sprintf("remote_user: %s\n", host.RemoteUser))
	sb.WriteString(fmt.Sprintf("deploy_path: %s\n", host.DeployPath))
	if host.SSHKey != "" {
		sb.WriteString(fmt.Sprintf("ssh_key: %s\n", host.SSHKey))
	}
	if host.SSHMultiplexing {
		sb.WriteString(fmt.Sprintf("ssh_multiplexing: %t\n", host.SSHMultiplexing))
	}
	if len(host.SSHOptions) > 0 {
		sb.WriteString("ssh_options:\n")
		for key, value := range host.SSHOptions {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// Commands
	if len(cfg.Commands) > 0 {
		sb.WriteString("\n# Commands\n")
		sb.WriteString("commands:\n")
		for _, cmd := range cfg.Commands {
			sb.WriteString(fmt.Sprintf("  - name: %s\n", cmd.Name))
			sb.WriteString(fmt.Sprintf("    run: %s\n", cmd.Run))
		}
	}

	return sb.String()
}

func formatStringSlice(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	if len(slice) == 1 {
		return fmt.Sprintf("[\"%s\"]", slice[0])
	}
	return fmt.Sprintf("[\"%s\"]", strings.Join(slice, "\", \""))
}

