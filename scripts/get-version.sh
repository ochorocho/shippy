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

# Try to get version from git tag
if git describe --tags --exact-match 2>/dev/null; then
    # On a tagged commit, return tag without 'v' prefix
    git describe --tags --exact-match 2>/dev/null | sed 's/^v//'
elif git describe --tags 2>/dev/null; then
    # Near a tag, show tag-commits-hash format
    git describe --tags 2>/dev/null | sed 's/^v//'
else
    # Fallback: branch name + short commit hash
    BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    echo "${BRANCH} (commit: ${COMMIT})"
fi
