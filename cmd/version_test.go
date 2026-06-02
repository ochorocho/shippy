package cmd

import (
	"runtime/debug"
	"testing"
)

func TestResolveBuildInfo(t *testing.T) {
	tests := []struct {
		name                  string
		version, commit, date string
		info                  *debug.BuildInfo
		wantVersion           string
		wantCommit            string
		wantDate              string
	}{
		{
			name:        "ldflags values take precedence over build info",
			version:     "0.0.5",
			commit:      "abc123",
			date:        "2026-06-02 20:11:43 UTC",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}},
			wantVersion: "0.0.5",
			wantCommit:  "abc123",
			wantDate:    "2026-06-02 20:11:43 UTC",
		},
		{
			name:        "go install at a tag reports the clean version",
			version:     "dev",
			commit:      "unknown",
			date:        "unknown",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v0.0.7"}},
			wantVersion: "0.0.7",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
		{
			name:    "local build collapses pseudo-version to <tag>-dev and fills vcs",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v0.0.5-0.20260602200117-dee248a22e74+dirty"},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "dee248a22e74ca9d8f5dcda2ce9436244b2975d4"},
					{Key: "vcs.time", Value: "2026-06-02T20:01:17Z"},
					{Key: "vcs.modified", Value: "true"},
				},
			},
			wantVersion: "0.0.5-dev",
			wantCommit:  "dee248a22e74ca9d8f5dcda2ce9436244b2975d4",
			wantDate:    "2026-06-02T20:01:17Z",
		},
		{
			name:    "pseudo-version with prerelease base",
			version: "dev",
			commit:  "unknown",
			date:    "unknown",
			info: &debug.BuildInfo{
				Main: debug.Module{Version: "v1.2.0-pre.0.20260602200117-dee248a22e74"},
			},
			wantVersion: "1.2.0-pre-dev",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
		{
			name:        "devel with no vcs keeps the dev sentinel",
			version:     "dev",
			commit:      "unknown",
			date:        "unknown",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantVersion: "dev",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
		{
			name:        "nil build info leaves values untouched",
			version:     "dev",
			commit:      "unknown",
			date:        "unknown",
			info:        nil,
			wantVersion: "dev",
			wantCommit:  "unknown",
			wantDate:    "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCommit, gotDate := resolveBuildInfo(tt.version, tt.commit, tt.date, tt.info)
			if gotVersion != tt.wantVersion {
				t.Errorf("version = %q, want %q", gotVersion, tt.wantVersion)
			}
			if gotCommit != tt.wantCommit {
				t.Errorf("commit = %q, want %q", gotCommit, tt.wantCommit)
			}
			if gotDate != tt.wantDate {
				t.Errorf("date = %q, want %q", gotDate, tt.wantDate)
			}
		})
	}
}
