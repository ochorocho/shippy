package cmd

import (
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Version is set via ldflags during build (see Makefile). When it is not
	// injected — e.g. `go install github.com/ochorocho/shippy@latest` or a
	// plain `go build` — it is derived from the embedded build info instead.
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

// pseudoTail matches the "-<timestamp>-<commit>" suffix of a Go pseudo-version,
// e.g. the ".20260602200117-dee248a22e74" in "0.0.5-0.20260602200117-dee248a22e74".
var pseudoTail = regexp.MustCompile(`[.-]\d{14}-[0-9a-f]{12}$`)

// resolveBuildInfo fills in version metadata from Go's embedded build info for
// builds that did not inject it via -ldflags. ldflags-provided values always
// take precedence; this only replaces the sentinel defaults.
//
// It is pure (no globals) so it can be unit-tested:
//   - `go install <module>@<tag>` reports a clean tag      -> "0.0.5"
//   - a local/untagged build reports a pseudo-version      -> "0.0.5-dev"
//   - vcs.* settings provide the commit hash and build time
func resolveBuildInfo(version, commit, date string, info *debug.BuildInfo) (string, string, string) {
	if info == nil {
		return version, commit, date
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if commit == "unknown" && setting.Value != "" {
				commit = setting.Value
			}
		case "vcs.time":
			if date == "unknown" && setting.Value != "" {
				date = setting.Value
			}
		}
	}

	// Only derive a version when ldflags did not inject one.
	if version != "dev" {
		return version, commit, date
	}

	mod := strings.TrimPrefix(info.Main.Version, "v")
	if i := strings.IndexByte(mod, '+'); i >= 0 { // drop +dirty / +incompatible
		mod = mod[:i]
	}

	switch {
	case mod == "" || mod == "(devel)":
		// No module version available; keep the "dev" sentinel.
	case pseudoTail.MatchString(mod):
		// Local/untagged build: collapse the pseudo-version to "<tag>-dev",
		// dropping the timestamp and short commit hash from the version string.
		base := pseudoTail.ReplaceAllString(mod, "")
		base = strings.TrimSuffix(base, "-0")
		base = strings.TrimSuffix(base, ".0")
		version = base + "-dev"
	default:
		// Installed at a real tag, e.g. `go install ...@v0.0.5`.
		version = mod
	}

	return version, commit, date
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Display the version, git commit, build date, and Go version of shippy.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)

	if info, ok := debug.ReadBuildInfo(); ok {
		Version, GitCommit, BuildDate = resolveBuildInfo(Version, GitCommit, BuildDate, info)
	}
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("shippy version %s\n", Version)
	fmt.Printf("  commit: %s\n", GitCommit)
	fmt.Printf("  built: %s\n", BuildDate)
	fmt.Printf("  go: %s\n", runtime.Version())
}
