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
  version "0.0.11"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.11/shippy-darwin-arm64"
      sha256 "26155c7057976f37f255eb474d5b127fcc5e8606ac1e8d2577f0cb8ec76855d3"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.11/shippy-darwin-amd64"
      sha256 "c45fba82687e3cf2fd6ed3367c6051f94988ef003b9a56e232886858510910e4"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.11/shippy-linux-arm64"
      sha256 "f37d8d54cbd90592178e65187b3be8742571b2e3020c016c63d17a1fbf4dccf2"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.11/shippy-linux-amd64"
      sha256 "60836fbfb55acb190c1d1c2cb73a781293e8e2f39bc470f6a1f50224e2ec0738"
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
