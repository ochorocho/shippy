package ssh

import (
	"strconv"

	sshconfig "github.com/kevinburke/ssh_config"
)

// applySSHConfig reads ~/.ssh/config and fills in missing ClientOptions fields.
// Values already set in .shippy.yaml always take precedence.
func applySSHConfig(opts *ClientOptions) {
	applyConfig(opts, sshconfig.Get)
}

// applyConfig fills in missing ClientOptions fields using the provided getter.
// Extracted for testability; production callers pass sshconfig.Get.
func applyConfig(opts *ClientOptions, get func(alias, key string) string) {
	alias := opts.Host

	if opts.ProxyCommand == "" {
		if cmd := get(alias, "ProxyCommand"); cmd != "" {
			opts.ProxyCommand = cmd
		}
	}

	// Port: only apply when not set by .shippy.yaml
	if opts.Port == 0 {
		if portStr := get(alias, "Port"); portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil && port > 0 {
				opts.Port = port
			}
		}
	}

	if opts.User == "" {
		if user := get(alias, "User"); user != "" {
			opts.User = user
		}
	}

	if opts.KeyPath == "" {
		if identityFile := get(alias, "IdentityFile"); identityFile != "" {
			opts.KeyPath = identityFile
		}
	}
}
