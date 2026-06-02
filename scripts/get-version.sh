#!/bin/bash
set -e

# Version detection priority:
# 1. SHIPPY_VERSION environment variable (for custom builds)
# 2. Git tag (if on a tagged commit)
# 3. Fallback: branch name + commit hash

if [ -n "$SHIPPY_VERSION" ]; then
    echo "$SHIPPY_VERSION"
    exit 0
fi

# Try to get version from git tag.
# NB: the command in an `if` condition still writes its stdout, so the describe
# output must be suppressed there (>/dev/null) — otherwise the tag leaks into
# the result and gets doubled with the value printed in the body.
if git describe --tags --exact-match >/dev/null 2>&1; then
    # On a tagged commit (release build): return that tag without the 'v' prefix.
    git describe --tags --exact-match 2>/dev/null | sed 's/^v//'
else
    # Dev build: base the version on the HIGHEST release tag in the repo
    # (regardless of branch ancestry) and mark it "-dev". Requires the tags to
    # be present locally (CI fetches them via fetch-depth: 0).
    LATEST=$(git tag --sort=-v:refname 2>/dev/null | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 | sed 's/^v//')
    if [ -n "$LATEST" ]; then
        echo "${LATEST}-dev"
    else
        # No release tags at all: fall back to branch name + short commit hash.
        BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
        COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
        echo "${BRANCH} (commit: ${COMMIT})"
    fi
fi
