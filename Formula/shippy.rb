# Homebrew formula for shippy (prebuilt-binary install).
#
# Install via tap:
#   brew tap ochorocho/shippy https://github.com/ochorocho/shippy
#   brew install shippy
#
# Do NOT hand-edit the version/url/sha256 values below — run `make brew-formula`
# (or scripts/update-formula.sh) to bump them for a release; the Release
# workflow does this automatically on tagged builds.
class Shippy < Formula
  desc "Zero-downtime deployment tool for Composer based PHP projects"
  homepage "https://github.com/ochorocho/shippy"
  version "0.0.6"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.6/shippy-darwin-arm64"
      sha256 "3fc0e30d1e761e69815bc2c127bae4d8ecdcc56ff9d462d34e32cb65488cba51"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.6/shippy-darwin-amd64"
      sha256 "e560abd1c453b868d2481c7df9fb88605ca5831bc46a18e83fce2bd994b4083c"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.6/shippy-linux-arm64"
      sha256 "3b780a3e5268ac1ec6cff0eecf6713528f998e69ea0773cb5ed3a613a660ace4"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.6/shippy-linux-amd64"
      sha256 "c65a9f59921760d2aaa00cfe535a5fccfb706a9287b547fc3bc6054d7ade4374"
    end
  end

  def install
    # Exactly one prebuilt binary is staged for the host platform; rename to `shippy`.
    binary = Dir["*"].find { |f| File.file?(f) }
    bin.install binary => "shippy"
  end

  test do
    assert_match "shippy version", shell_output("#{bin}/shippy version")
    assert_match "Usage:", shell_output("#{bin}/shippy --help")
  end
end
