#!/bin/bash
set -euo pipefail

# Update the Homebrew formula (prebuilt-binary install) to a given release tag:
# rewrites `version`, each per-arch download `url`, and the matching `sha256`.
#
# Usage: scripts/update-formula.sh [vX.Y.Z]
#   Defaults to the most recent v*.*.* git tag.
#
# Checksums are read from the `*.checksum` assets attached to the release, so
# the (large) binaries themselves are never downloaded. Kept POSIX/bash-3.2
# compatible (no associative arrays) so it runs on stock macOS bash too.

REPO="ochorocho/shippy"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FORMULA="$ROOT/Formula/shippy.rb"

TAG="${1:-}"
if [ -z "$TAG" ]; then
    TAG=$(git tag --sort=-creatordate | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | head -1 || true)
fi
if [ -z "$TAG" ]; then
    echo "No v*.*.* tag found" >&2
    exit 1
fi
VERSION="${TAG#v}"

# Fetch and validate the published checksum for one architecture's binary.
# Retries a few times: just-uploaded release assets can take a moment to
# become downloadable from the releases/download/ CDN path.
fetch_sum() {
    arch="$1"
    url="https://github.com/${REPO}/releases/download/${TAG}/shippy-${arch}.checksum"
    attempt=1
    while [ "$attempt" -le 5 ]; do
        echo "Fetching checksum (attempt ${attempt}): ${url}" >&2
        sum=$(curl -fsSL "$url" 2>/dev/null | awk '{print $1}' || true)
        if printf '%s' "$sum" | grep -Eq '^[0-9a-f]{64}$'; then
            printf '%s' "$sum"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 5
    done
    echo "Could not fetch a valid checksum for ${arch} from ${url}" >&2
    exit 1
}

SHA_DARWIN_AMD64=$(fetch_sum darwin-amd64)
SHA_DARWIN_ARM64=$(fetch_sum darwin-arm64)
SHA_LINUX_AMD64=$(fetch_sum linux-amd64)
SHA_LINUX_ARM64=$(fetch_sum linux-arm64)

# Rewrite version, each url's tag segment, and the sha256 line that follows it.
tmp=$(mktemp)
awk -v tag="$TAG" -v version="$VERSION" \
    -v s_darwin_amd64="$SHA_DARWIN_AMD64" \
    -v s_darwin_arm64="$SHA_DARWIN_ARM64" \
    -v s_linux_amd64="$SHA_LINUX_AMD64" \
    -v s_linux_arm64="$SHA_LINUX_ARM64" '
    /^  version "/ { sub(/"[^"]*"/, "\"" version "\""); print; next }
    /^      url "/ {
        gsub(/download\/v[0-9]+\.[0-9]+\.[0-9]+\//, "download/" tag "/")
        if      ($0 ~ /shippy-darwin-amd64"/) pending = "darwin_amd64"
        else if ($0 ~ /shippy-darwin-arm64"/) pending = "darwin_arm64"
        else if ($0 ~ /shippy-linux-amd64"/)  pending = "linux_amd64"
        else if ($0 ~ /shippy-linux-arm64"/)  pending = "linux_arm64"
        print; next
    }
    /^      sha256 "/ {
        if      (pending == "darwin_amd64") sub(/"[0-9a-f]*"/, "\"" s_darwin_amd64 "\"")
        else if (pending == "darwin_arm64") sub(/"[0-9a-f]*"/, "\"" s_darwin_arm64 "\"")
        else if (pending == "linux_amd64")  sub(/"[0-9a-f]*"/, "\"" s_linux_amd64 "\"")
        else if (pending == "linux_arm64")  sub(/"[0-9a-f]*"/, "\"" s_linux_arm64 "\"")
        pending = ""
        print; next
    }
    { print }
' "$FORMULA" > "$tmp"
mv "$tmp" "$FORMULA"
chmod a+r "$FORMULA"

echo "Updated ${FORMULA} -> ${TAG}" >&2
