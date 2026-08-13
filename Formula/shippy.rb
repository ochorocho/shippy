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
  version "0.1.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.1/shippy-darwin-arm64"
      sha256 "6df447419d170b899546b64eeaa83789f3016391a9afafa4402bc615d9d182a1"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.1/shippy-darwin-amd64"
      sha256 "352cb77ed1b4020159803c59fbd47770f31ad7cccc6aa9855ff61bca89e04ac6"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.1/shippy-linux-arm64"
      sha256 "3c27f1a17cbe14497877ea459577ecb5ba99c6c2d44c6c34b5ec130f59810de1"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.1/shippy-linux-amd64"
      sha256 "83d1be277615cb00735b4fa502547376c3ec625d9e00e25f046a23d95a7b9055"
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
