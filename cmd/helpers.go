package cmd

import (
	"errors"
	"io/fs"
	"sort"

	"github.com/ochorocho/shippy/internal/composer"
	"github.com/ochorocho/shippy/internal/config"
	"github.com/ochorocho/shippy/internal/ui"
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

// loadComposer loads the configured composer.json and reports whether it exists.
// A missing file only warns, as template variables without a fallback still fail
// when they are processed. A malformed file is an error.
func loadComposer(cfg *config.Config, out *ui.Output) (*composer.Composer, bool, error) {
	path := cfg.GetComposerPath()
	comp, err := composer.Parse(path)
	if errors.Is(err, fs.ErrNotExist) {
		out.Warning("composer.json not found at %s, continuing without it", path)
		return composer.Empty(), false, nil
	}
	if err != nil {
		out.Error("Failed to load composer.json from %s: %v", path, err)
		return nil, false, err
	}
	return comp, true, nil
}
