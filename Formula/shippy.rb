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
  version "0.0.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-darwin-arm64"
      sha256 "ad17f452b79dd488ea6354f6fd8bdb55af2f080ccecd276e39b9ee1c6ac5771a"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-darwin-amd64"
      sha256 "6c978669e21abb0ef3ff041e01d358e28ac48b0d61f1a780d1a2d98af0f355a0"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-linux-arm64"
      sha256 "6aefd2bdf81da802fe281c91caebe955f545d2cf88007ded3f44329245a512e8"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-linux-amd64"
      sha256 "ad5d7875e8ae626307002c576d2f21bbadf1fe56be9b14ce776de4f77bff5e49"
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
