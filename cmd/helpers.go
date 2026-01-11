package cmd

import (
	"sort"

	"tinnie/internal/config"
	"tinnie/internal/ui"
)

// selectHostFromArgs returns the host from args or prompts for interactive selection
func selectHostFromArgs(cfg *config.Config, args []string, out *ui.Output) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}

	hosts := make([]string, 0, len(cfg.Hosts))
	for name := range cfg.Hosts {
		hosts = append(hosts, name)
	}
	sort.Strings(hosts)

	selected, err := out.SelectHost(hosts)
	if err != nil {
		out.Error("Host selection failed: %v", err)
		return "", err
	}
	return selected, nil
}

// loadConfigFile loads and returns the configuration file with error handling
func loadConfigFile(out *ui.Output) (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		out.Error("Failed to load config: %v", err)
		return nil, err
	}
	return cfg, nil
}
