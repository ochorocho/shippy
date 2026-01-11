package cmd

import (
	"fmt"
	"strings"

	"shippy/internal/composer"
	"shippy/internal/config"
	"shippy/internal/ui"

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

var showCmd = &cobra.Command{
	Use:   "show [host]",
	Short: "Show the complete resolved configuration",
	Long: `Display the complete configuration with all defaults applied and templates resolved.

If a host is specified, shows the effective configuration for that host
including global settings and per-host overrides.

Examples:
  shippy config show              # Show complete config
  shippy config show production   # Show config for production host`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(validateCmd)
	configCmd.AddCommand(showCmd)
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

func runShow(cmd *cobra.Command, args []string) error {
	out := ui.New()

	// Load config
	cfg, err := config.Load(cfgFile)
	if err != nil {
		out.Error("Failed to load config: %v", err)
		return err
	}

	// Load composer.json for template processing using configured path
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

	// Apply defaults
	cfg.ApplyDefaults()

	// If host specified, show host-specific resolved config
	if len(args) > 0 {
		hostName := args[0]
		host, err := cfg.GetHost(hostName)
		if err != nil {
			out.Error("Host not found: %v", err)
			return err
		}

		return showHostConfig(cfg, host, hostName, out)
	}

	// Show complete config
	return showCompleteConfig(cfg, out)
}

func showCompleteConfig(cfg *config.Config, out *ui.Output) error {
	out.Header("Complete Configuration (with defaults applied)")
	fmt.Println()

	// Convert to YAML with syntax highlighting
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	printYAMLWithHighlighting(string(data))
	return nil
}

func showHostConfig(cfg *config.Config, host *config.Host, hostName string, out *ui.Output) error {
	out.Header(fmt.Sprintf("Configuration for host: %s", hostName))
	fmt.Println()

	// Build resolved config for this host
	resolvedConfig := buildResolvedHostConfig(cfg, host, hostName)

	printYAMLWithHighlighting(resolvedConfig)
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

func printYAMLWithHighlighting(yamlContent string) {
	lines := strings.Split(yamlContent, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			fmt.Println()
			continue
		}

		// Comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			fmt.Printf("\033[90m%s\033[0m\n", line) // Gray
			continue
		}

		// Keys and values
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}

			// Color the key
			indent := len(key) - len(strings.TrimLeft(key, " "))
			keyTrimmed := strings.TrimSpace(key)

			fmt.Printf("%s\033[36m%s\033[0m:", strings.Repeat(" ", indent), keyTrimmed) // Cyan for keys

			// Color the value
			if value != "" {
				valueTrimmed := strings.TrimSpace(value)

				// Booleans
				if valueTrimmed == "true" || valueTrimmed == "false" {
					fmt.Printf(" \033[33m%s\033[0m\n", valueTrimmed) // Yellow for booleans
				// Numbers
				} else if isNumber(valueTrimmed) {
					fmt.Printf(" \033[35m%s\033[0m\n", valueTrimmed) // Magenta for numbers
				// Strings
				} else {
					fmt.Printf(" \033[32m%s\033[0m\n", valueTrimmed) // Green for strings
				}
			} else {
				fmt.Println()
			}
			continue
		}

		// List items
		if strings.Contains(line, "- ") {
			parts := strings.SplitN(line, "- ", 2)
			indent := parts[0]
			value := ""
			if len(parts) > 1 {
				value = parts[1]
			}
			fmt.Printf("%s\033[90m-\033[0m \033[32m%s\033[0m\n", indent, value) // Gray dash, green value
			continue
		}

		// Default
		fmt.Println(line)
	}
}

func isNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}
