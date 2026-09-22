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
  version "0.2.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-darwin-arm64"
      sha256 "4ad88b0667c46114793afbf958c80621c721fb10920b052d05d2f9dc74acecdf"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-darwin-amd64"
      sha256 "3a6e9cbb3133fba863fe91e1e8e0da1a0e0b048b7fddc7361a42cbf0cdff18b8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-linux-arm64"
      sha256 "0e747070a0b6a7e829fb19f81b3b3c5de7625055789b18e362334611f26096c4"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-linux-amd64"
      sha256 "3b5a76a4cd1044c383307d9ebe6d4e241015ffb70a5ba7153a6264a154b22e1b"
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
