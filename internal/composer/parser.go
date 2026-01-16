package composer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Composer represents the composer.json structure
type Composer struct {
	data map[string]interface{}
}

// Parse reads and parses a composer.json file
func Parse(path string) (*Composer, error) {
	// #nosec G304 -- Composer path comes from config or project root
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read composer.json: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse composer.json: %w", err)
	}

	return &Composer{data: result}, nil
}

// Get retrieves a value from composer.json using dot notation
// Example: "name" or "extra.typo3/cms.web-dir"
func (c *Composer) Get(key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	var current interface{} = c.data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			val, ok := v[part]
			if !ok {
				return nil, fmt.Errorf("key '%s' not found in composer.json", key)
			}
			current = val
		default:
			return nil, fmt.Errorf("cannot traverse into non-object at key '%s'", part)
		}
	}

	return current, nil
}

// GetString retrieves a string value from composer.json
func (c *Composer) GetString(key string) (string, error) {
	val, err := c.Get(key)
	if err != nil {
		return "", err
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key '%s' is not a string", key)
	}

	return str, nil
}

// GetData returns the entire composer.json data as a map
func (c *Composer) GetData() map[string]interface{} {
	return c.data
}
