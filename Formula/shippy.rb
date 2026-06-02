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
  version "0.0.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.3/shippy-darwin-arm64"
      sha256 "cc2ab787988833184963b42e4107a6d044aca79e9b0f483484529110e6f455ee"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.3/shippy-darwin-amd64"
      sha256 "23c34728ca381dc2ad1a2b090d8d0df9c33d7a396eb55a8343f0961231e8f619"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.3/shippy-linux-arm64"
      sha256 "7f96c2f8f6c7e01c51f8e248f95ef846b3276faf69b57adf252da912345d5fbf"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.3/shippy-linux-amd64"
      sha256 "fd4dca9b72ce8c99ec4e334613953d07937a20dc581588ff48b7a3eded4e8098"
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
