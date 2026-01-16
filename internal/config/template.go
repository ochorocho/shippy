package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"tinnie/internal/composer"
)

var (
	templateVarRegex = regexp.MustCompile(`\{\{([^}]+)\}\}`)
	envVarRegex      = regexp.MustCompile(`\$\{([^}]+)\}`)
)

// ProcessTemplates replaces template variables in the config with values from composer.json
func (c *Config) ProcessTemplates(comp *composer.Composer) error {
	var err error

	// Process global RsyncSrc
	c.RsyncSrc, err = replaceTemplateVars(c.RsyncSrc, comp)
	if err != nil {
		return fmt.Errorf("global rsync_src: %w", err)
	}

	for hostName, host := range c.Hosts {

		host.Hostname, err = replaceTemplateVars(host.Hostname, comp)
		if err != nil {
			return fmt.Errorf("host '%s'.hostname: %w", hostName, err)
		}

		host.RemoteUser, err = replaceTemplateVars(host.RemoteUser, comp)
		if err != nil {
			return fmt.Errorf("host '%s'.remote_user: %w", hostName, err)
		}

		host.DeployPath, err = replaceTemplateVars(host.DeployPath, comp)
		if err != nil {
			return fmt.Errorf("host '%s'.deploy_path: %w", hostName, err)
		}

		host.RsyncSrc, err = replaceTemplateVars(host.RsyncSrc, comp)
		if err != nil {
			return fmt.Errorf("host '%s'.rsync_src: %w", hostName, err)
		}

		host.SSHKey, err = replaceTemplateVars(host.SSHKey, comp)
		if err != nil {
			return fmt.Errorf("host '%s'.ssh_key: %w", hostName, err)
		}

		// Process SSH options
		for optName, optValue := range host.SSHOptions {
			processedValue, err := replaceTemplateVars(optValue, comp)
			if err != nil {
				return fmt.Errorf("host '%s'.ssh_options.%s: %w", hostName, optName, err)
			}
			host.SSHOptions[optName] = processedValue
		}

		// Process exclude patterns
		for i, pattern := range host.Exclude {
			host.Exclude[i], err = replaceTemplateVars(pattern, comp)
			if err != nil {
				return fmt.Errorf("host '%s'.exclude[%d]: %w", hostName, i, err)
			}
		}

		// Process include patterns
		for i, pattern := range host.Include {
			host.Include[i], err = replaceTemplateVars(pattern, comp)
			if err != nil {
				return fmt.Errorf("host '%s'.include[%d]: %w", hostName, i, err)
			}
		}

		c.Hosts[hostName] = host
	}

	// Process commands
	for i, cmd := range c.Commands {
		var err error

		c.Commands[i].Name, err = replaceTemplateVars(cmd.Name, comp)
		if err != nil {
			return fmt.Errorf("command[%d].name: %w", i, err)
		}

		c.Commands[i].Run, err = replaceTemplateVars(cmd.Run, comp)
		if err != nil {
			return fmt.Errorf("command[%d].run: %w", i, err)
		}
	}

	return nil
}

// replaceTemplateVars replaces all {{key.path}} or {{key.path|fallback}} variables and ${ENV_VAR} or ${ENV_VAR|fallback} in the string
func replaceTemplateVars(input string, comp *composer.Composer) (string, error) {
	var errors []string

	// First, replace composer.json template variables {{key}}
	result := templateVarRegex.ReplaceAllStringFunc(input, func(match string) string {
		// Extract the key from {{key}} or {{key|fallback}}
		content := strings.TrimSpace(match[2 : len(match)-2])

		// Check for fallback value (syntax: {{key|fallback}})
		var key, fallback string
		if idx := strings.Index(content, "|"); idx != -1 {
			key = strings.TrimSpace(content[:idx])
			fallback = strings.TrimSpace(content[idx+1:])
		} else {
			key = content
		}

		// Get value from composer.json
		val, err := comp.Get(key)
		if err != nil {
			// If fallback is specified, use it
			if fallback != "" {
				return fallback
			}
			// Otherwise, report error
			errors = append(errors, fmt.Sprintf("template variable '{{%s}}': %s", key, err.Error()))
			return match // Return original if error
		}

		// Convert to string
		return fmt.Sprintf("%v", val)
	})

	// Second, replace environment variables ${ENV_VAR}
	result = envVarRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the variable name from ${VAR} or ${VAR|fallback}
		content := strings.TrimSpace(match[2 : len(match)-1])

		// Check for fallback value (syntax: ${VAR|fallback})
		var varName, fallback string
		if idx := strings.Index(content, "|"); idx != -1 {
			varName = strings.TrimSpace(content[:idx])
			fallback = strings.TrimSpace(content[idx+1:])
		} else {
			varName = content
		}

		// Get value from environment
		val, exists := os.LookupEnv(varName)
		if !exists {
			// If fallback is specified, use it
			if fallback != "" {
				return fallback
			}
			// Otherwise, report error
			errors = append(errors, fmt.Sprintf("environment variable '${%s}' is not set", varName))
			return match // Return original if error
		}

		// Return environment variable value (even if empty string)
		return val
	})

	if len(errors) > 0 {
		return "", fmt.Errorf("%s", strings.Join(errors, "; "))
	}

	return result, nil
}
