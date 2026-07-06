package config

import "testing"

func strPtr(s string) *string { return &s }

func TestGetCommandContext(t *testing.T) {
	tests := []struct {
		name   string
		global string
		host   *Host
		cmd    Command
		want   string
	}{
		{
			name: "nothing set runs on host",
			cmd:  Command{Run: "x"},
			want: "",
		},
		{
			name:   "global applies when host and command unset",
			global: "docker exec php85",
			host:   &Host{},
			cmd:    Command{Run: "x"},
			want:   "docker exec php85",
		},
		{
			name:   "host overrides global",
			global: "docker exec php85",
			host:   &Host{CommandContext: "docker exec php84"},
			cmd:    Command{Run: "x"},
			want:   "docker exec php84",
		},
		{
			name:   "command overrides host and global",
			global: "docker exec php85",
			host:   &Host{CommandContext: "docker exec php84"},
			cmd:    Command{Run: "x", CommandContext: strPtr("podman exec php83")},
			want:   "podman exec php83",
		},
		{
			name:   "empty command override forces run on host despite host/global",
			global: "docker exec php85",
			host:   &Host{CommandContext: "docker exec php84"},
			cmd:    Command{Run: "x", CommandContext: strPtr("")},
			want:   "",
		},
		{
			name:   "nil host falls back to global",
			global: "docker exec php85",
			host:   nil,
			cmd:    Command{Run: "x"},
			want:   "docker exec php85",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{CommandContext: tt.global}
			got := c.GetCommandContext(tt.host, tt.cmd)
			if got != tt.want {
				t.Errorf("GetCommandContext() = %q, want %q", got, tt.want)
			}
		})
	}
}
