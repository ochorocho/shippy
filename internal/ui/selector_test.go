package ui

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// typedKeys delivers one byte per read. Bubble Tea treats runes read together as
// a single key press, so a plain reader would not behave like someone typing.
type typedKeys struct{ input string }

func (k *typedKeys) Read(p []byte) (int, error) {
	if k.input == "" {
		return 0, io.EOF
	}
	p[0] = k.input[0]
	k.input = k.input[1:]
	return 1, nil
}

// keys returns program options that feed the given keystrokes to a selector.
func keys(t *testing.T, input string) []tea.ProgramOption {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return []tea.ProgramOption{
		tea.WithContext(ctx),
		tea.WithInput(&typedKeys{input: input}),
		tea.WithOutput(io.Discard),
		tea.WithoutSignalHandler(),
	}
}

// Newest first, as rollback lists them.
func testReleases() []ReleaseItem {
	return []ReleaseItem{
		{Name: "v3", IsCurrent: true},
		{Name: "v2"},
		{Name: "v1"},
	}
}

func TestReleaseSelectorSelection(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"enter picks the newest non-current release", "\r", "v2"},
		{"j moves to an older release", "j\r", "v1"},
		{"k moves back to a newer release", "jk\r", "v2"},
		{"k cannot reach the current release", "kkk\r", "v2"},
		{"j stops at the oldest release", "jjj\r", "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := runReleaseSelector(testReleases(), keys(t, tt.input)...)
			if err != nil {
				t.Fatalf("selector failed: %v", err)
			}
			if got.Name != tt.want {
				t.Errorf("selected %q, want %q", got.Name, tt.want)
			}
		})
	}
}

func TestReleaseSelectorCancel(t *testing.T) {
	got, err := runReleaseSelector(testReleases(), keys(t, "jq")...)
	if err == nil || !strings.Contains(err.Error(), "selection cancelled") {
		t.Fatalf("err = %v, want selection cancelled", err)
	}
	if got != nil {
		t.Errorf("selected %q, want nothing", got.Name)
	}
}

func TestHostSelectorSelection(t *testing.T) {
	got, err := runHostSelector([]string{"production", "staging"}, keys(t, "j\r")...)
	if err != nil {
		t.Fatalf("selector failed: %v", err)
	}
	if got != "staging" {
		t.Errorf("selected %q, want staging", got)
	}
}

func TestHostSelectorCancel(t *testing.T) {
	_, err := runHostSelector([]string{"production", "staging"}, keys(t, "q")...)
	if err == nil || !strings.Contains(err.Error(), "selection cancelled") {
		t.Fatalf("err = %v, want selection cancelled", err)
	}
}
