package config

import "testing"

func TestGetFileMode(t *testing.T) {
	tests := []struct {
		name   string
		global string
		host   *Host
		want   string
	}{
		{name: "unset returns default", host: &Host{}, want: DefaultFileMode},
		{name: "global applies when host unset", global: "0640", host: &Host{}, want: "0640"},
		{name: "host overrides global", global: "0640", host: &Host{FileMode: "0600"}, want: "0600"},
		{name: "host applies when global unset", host: &Host{FileMode: "0600"}, want: "0600"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{FileMode: tt.global}
			if got := c.GetFileMode(tt.host); got != tt.want {
				t.Errorf("GetFileMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetDirMode(t *testing.T) {
	tests := []struct {
		name   string
		global string
		host   *Host
		want   string
	}{
		{name: "unset returns default", host: &Host{}, want: DefaultDirMode},
		{name: "global applies when host unset", global: "0755", host: &Host{}, want: "0755"},
		{name: "host overrides global", global: "0755", host: &Host{DirMode: "2750"}, want: "2750"},
		{name: "host applies when global unset", host: &Host{DirMode: "2750"}, want: "2750"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{DirMode: tt.global}
			if got := c.GetDirMode(tt.host); got != tt.want {
				t.Errorf("GetDirMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		want    int64
		wantErr bool
	}{
		{name: "file mode", mode: "0644", want: 0o644},
		{name: "dir mode with setgid", mode: "2755", want: 0o2755},
		{name: "without leading zero", mode: "755", want: 0o755},
		{name: "non-octal digit", mode: "0678", wantErr: true},
		{name: "not a number", mode: "rwxr-xr-x", wantErr: true},
		{name: "empty", mode: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMode(tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseMode(%q) expected error, got %o", tt.mode, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseMode(%q) unexpected error: %v", tt.mode, err)
			}
			if got != tt.want {
				t.Errorf("ParseMode(%q) = %o, want %o", tt.mode, got, tt.want)
			}
		})
	}
}

func TestValidateRejectsBadMode(t *testing.T) {
	c := &Config{
		FileMode: "0678", // invalid octal
		Hosts: map[string]Host{
			"staging": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
		},
	}
	if err := c.Validate(); err == nil {
		t.Error("Validate() should reject an invalid file_mode")
	}
}
