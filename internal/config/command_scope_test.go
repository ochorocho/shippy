package config

import "testing"

func TestCommandAppliesToHost(t *testing.T) {
	tests := []struct {
		name     string
		cmd      Command
		hostName string
		want     bool
	}{
		{
			name:     "no filters runs everywhere",
			cmd:      Command{Run: "x"},
			hostName: "production",
			want:     true,
		},
		{
			name:     "only matches listed host",
			cmd:      Command{Run: "x", Only: []string{"production"}},
			hostName: "production",
			want:     true,
		},
		{
			name:     "only skips unlisted host",
			cmd:      Command{Run: "x", Only: []string{"production"}},
			hostName: "staging",
			want:     false,
		},
		{
			name:     "except skips listed host",
			cmd:      Command{Run: "x", Except: []string{"staging"}},
			hostName: "staging",
			want:     false,
		},
		{
			name:     "except runs on unlisted host",
			cmd:      Command{Run: "x", Except: []string{"staging"}},
			hostName: "production",
			want:     true,
		},
		{
			name:     "except wins even if only would include",
			cmd:      Command{Run: "x", Only: []string{"production"}, Except: []string{"production"}},
			hostName: "production",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.AppliesToHost(tt.hostName)
			if got != tt.want {
				t.Errorf("AppliesToHost(%q) = %v, want %v", tt.hostName, got, tt.want)
			}
		})
	}
}

func TestValidateCommandHostRefs(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "unknown host in only fails validation",
			cfg: &Config{
				Hosts: map[string]Host{
					"production": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
				},
				Commands: []Command{
					{Name: "c", Run: "x", Only: []string{"staging"}},
				},
			},
			wantErr: true,
		},
		{
			name: "unknown host in except fails validation",
			cfg: &Config{
				Hosts: map[string]Host{
					"production": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
				},
				Commands: []Command{
					{Name: "c", Run: "x", Except: []string{"staging"}},
				},
			},
			wantErr: true,
		},
		{
			name: "known host in only passes validation",
			cfg: &Config{
				Hosts: map[string]Host{
					"production": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
				},
				Commands: []Command{
					{Name: "c", Run: "x", Only: []string{"production"}},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
