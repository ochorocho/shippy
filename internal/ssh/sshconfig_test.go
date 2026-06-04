package ssh

import "testing"

// testGetter returns a getter backed by a static map, simulating ~/.ssh/config entries.
func testGetter(cfg map[string]map[string]string) func(string, string) string {
	return func(alias, key string) string {
		if host, ok := cfg[alias]; ok {
			return host[key]
		}
		return ""
	}
}

func TestApplyConfigFillsMissingValues(t *testing.T) {
	get := testGetter(map[string]map[string]string{
		"myhost": {
			"ProxyCommand": "/usr/bin/nc %h %p",
			"Port":         "2222",
			"User":         "admin",
			"IdentityFile": "/home/user/.ssh/mykey",
		},
	})

	opts := ClientOptions{Host: "myhost"}
	applyConfig(&opts, get)

	if opts.ProxyCommand != "/usr/bin/nc %h %p" {
		t.Errorf("ProxyCommand = %q, want %q", opts.ProxyCommand, "/usr/bin/nc %h %p")
	}
	if opts.Port != 2222 {
		t.Errorf("Port = %d, want 2222", opts.Port)
	}
	if opts.User != "admin" {
		t.Errorf("User = %q, want %q", opts.User, "admin")
	}
	if opts.KeyPath != "/home/user/.ssh/mykey" {
		t.Errorf("KeyPath = %q, want %q", opts.KeyPath, "/home/user/.ssh/mykey")
	}
}

func TestApplyConfigDoesNotOverwriteExistingValues(t *testing.T) {
	get := testGetter(map[string]map[string]string{
		"myhost": {
			"ProxyCommand": "config_cmd",
			"Port":         "2222",
			"User":         "config_user",
			"IdentityFile": "/config/key",
		},
	})

	opts := ClientOptions{
		Host:         "myhost",
		ProxyCommand: "shippy_cmd",
		Port:         22,
		User:         "shippy_user",
		KeyPath:      "/shippy/key",
	}
	applyConfig(&opts, get)

	if opts.ProxyCommand != "shippy_cmd" {
		t.Errorf("ProxyCommand overwritten: got %q, want %q", opts.ProxyCommand, "shippy_cmd")
	}
	if opts.Port != 22 {
		t.Errorf("Port overwritten: got %d, want 22", opts.Port)
	}
	if opts.User != "shippy_user" {
		t.Errorf("User overwritten: got %q, want %q", opts.User, "shippy_user")
	}
	if opts.KeyPath != "/shippy/key" {
		t.Errorf("KeyPath overwritten: got %q, want %q", opts.KeyPath, "/shippy/key")
	}
}

func TestApplyConfigNoMatchLeavesOptsUnchanged(t *testing.T) {
	get := testGetter(map[string]map[string]string{
		"otherhost": {"ProxyCommand": "some_cmd"},
	})

	opts := ClientOptions{Host: "myhost"}
	applyConfig(&opts, get)

	if opts.ProxyCommand != "" || opts.Port != 0 || opts.User != "" || opts.KeyPath != "" {
		t.Errorf("opts should be unchanged for unknown host, got: %+v", opts)
	}
}

func TestApplyConfigInvalidPortIgnored(t *testing.T) {
	get := testGetter(map[string]map[string]string{
		"myhost": {"Port": "notanumber"},
	})

	opts := ClientOptions{Host: "myhost"}
	applyConfig(&opts, get)

	if opts.Port != 0 {
		t.Errorf("invalid port should be ignored, got %d", opts.Port)
	}
}

func TestApplyConfigZeroPortFromConfig(t *testing.T) {
	get := testGetter(map[string]map[string]string{
		"myhost": {"Port": "0"},
	})

	opts := ClientOptions{Host: "myhost"}
	applyConfig(&opts, get)

	// Port 0 in config is treated as invalid (must be > 0)
	if opts.Port != 0 {
		t.Errorf("port 0 in config should be ignored, got %d", opts.Port)
	}
}
