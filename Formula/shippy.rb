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
  version "0.2.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.3/shippy-darwin-arm64"
      sha256 "387f3963ca4a181d3d29f91381d33e2a2bbd68ea22b154e4d09360062c43f559"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.3/shippy-darwin-amd64"
      sha256 "f9934a3dee5dec3918614b0384c1bb25e99239f96b67f75be0fec24b12d7712b"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.3/shippy-linux-arm64"
      sha256 "52a7fa290f54bae31f1b9289903c01ab65601e14b1d779986c90c80c15461b2b"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.3/shippy-linux-amd64"
      sha256 "763c0f68864a2a9915c14a6223973faf5bf6fdbced4a9f4777b35432a8b7cb46"
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
