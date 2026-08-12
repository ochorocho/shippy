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
  version "0.1.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.0/shippy-darwin-arm64"
      sha256 "90d2648863991033d955558ca983f2946193b1c63c2fdd634aff5713af2b02d0"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.0/shippy-darwin-amd64"
      sha256 "a5952acdfcdf77bce2f2ef27f5a1f27fc54031488e7700de7c343c2705a08f52"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.0/shippy-linux-arm64"
      sha256 "37d98a9fd439aec06dd204a879729df8165d3e3eebb02a6a7646fdc0c25fc083"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.0/shippy-linux-amd64"
      sha256 "1bcab93644c6c27786f90bb4fd19d2c2a1cb92d68bd27e33a6b947aec2398f99"
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
