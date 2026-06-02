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
  version "0.0.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.5/shippy-darwin-arm64"
      sha256 "dda6a949cf377e0a4a135a712af01590e489796f4e9168dbe60646cba6f88657"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.5/shippy-darwin-amd64"
      sha256 "f0de47d66b606f67bb9329ab09d31334cbf081b8b6da892d80509b5e417292b5"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.5/shippy-linux-arm64"
      sha256 "5fe59aa797b2f6f5992d3fb12213ade4ae2f760575f5f7c5662f71f9e44f89bd"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.5/shippy-linux-amd64"
      sha256 "8527d3aeae9eb0d08543397be364920cdb48d937d252b8b36736e3b66d9d8664"
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
