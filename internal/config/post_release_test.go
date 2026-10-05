package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ochorocho/shippy/internal/composer"
)

// emptyComposer returns a Composer parsed from an empty composer.json so template
// variables resolve via their |fallback values.
func emptyComposer(t *testing.T) *composer.Composer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "composer.json")
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	comp, err := composer.Parse(path)
	if err != nil {
		t.Fatalf("parse composer: %v", err)
	}
	return comp
}

func TestProcessTemplatesPostReleaseCommands(t *testing.T) {
	ctx := "docker exec php"
	c := &Config{
		CommandsPostRelease: []Command{
			{
				Name:           "Flush opcache",
				Run:            "./{{config.bin-dir|vendor/bin}}/typo3 cache:flush",
				CommandContext: &ctx,
			},
		},
	}

	if err := c.ProcessTemplates(emptyComposer(t)); err != nil {
		t.Fatalf("ProcessTemplates: %v", err)
	}

	got := c.CommandsPostRelease[0]
	if got.Run != "./vendor/bin/typo3 cache:flush" {
		t.Errorf("Run = %q, want templated path", got.Run)
	}
	if got.CommandContext == nil || *got.CommandContext != ctx {
		t.Errorf("CommandContext = %v, want %q", got.CommandContext, ctx)
	}
}

func TestValidateRejectsBadPostReleaseContext(t *testing.T) {
	bad := "ctx\x00injection" // control char rejected by validateShellSafe
	c := &Config{
		Hosts: map[string]Host{
			"staging": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
		},
		CommandsPostRelease: []Command{
			{Name: "x", Run: "echo x", CommandContext: &bad},
		},
	}
	if err := c.Validate(); err == nil {
		t.Error("Validate() should reject an unsafe commands_post_release context")
	}
}

func TestValidateRejectsUnknownPostReleaseHostRef(t *testing.T) {
	c := &Config{
		Hosts: map[string]Host{
			"staging": {Hostname: "h", RemoteUser: "u", DeployPath: "/p"},
		},
		CommandsPostRelease: []Command{
			{Name: "x", Run: "echo x", Only: []string{"nonexistent"}},
		},
	}
	if err := c.Validate(); err == nil {
		t.Error("Validate() should reject a commands_post_release only: ref to an unknown host")
	}
}

// TestPostReleaseCommandsHaveNoDefault confirms the list stays empty unless the
// user configures it (unlike Commands, which gets the default TYPO3 set).
func TestPostReleaseCommandsHaveNoDefault(t *testing.T) {
	c := &Config{}
	c.ApplyDefaults()
	if len(c.CommandsPostRelease) != 0 {
		t.Errorf("CommandsPostRelease got %d defaults, want 0", len(c.CommandsPostRelease))
	}
}
