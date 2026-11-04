package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyDefaultsCommands(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want int
	}{
		{"key missing uses defaults", "", 5},
		{"empty value uses defaults", "commands:\n", 5},
		{"empty list runs nothing", "commands: []\n", 0},
		{"own commands are kept", "commands:\n  - name: a\n    run: b\n", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), ".shippy.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}

			cfg.ApplyDefaults()

			if got := len(cfg.Commands); got != tt.want {
				t.Errorf("got %d commands, want %d", got, tt.want)
			}
		})
	}
}
