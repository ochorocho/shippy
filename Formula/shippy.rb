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
  version "0.1.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.4/shippy-darwin-arm64"
      sha256 "d8fb28be8c4fd372ab760f66571e6613e69701297b82e33b1c3dd1eb4a868da8"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.4/shippy-darwin-amd64"
      sha256 "4257190a0a8b5c590defce0b5ae4cd578e1525d12eadbec7412e91155ab06922"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.4/shippy-linux-arm64"
      sha256 "64a79d8c3f7d80d6afa088d7c5d883642da5e779f2d1605c512164899b505ead"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.4/shippy-linux-amd64"
      sha256 "d8faf6cfc9d14b088d7432e245a55439abd7592198f3ccbfc551f9ddbbd46347"
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
