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
  version "0.1.3"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.3/shippy-darwin-arm64"
      sha256 "c3f0a6de6505e0a5c3853bfe6d5b5dc5456b9127296e09d62807d09973d250a9"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.3/shippy-darwin-amd64"
      sha256 "78e0a39a5240c9d622b935c98ef4c5b672ea3c1919d7d5f81f649891769116e0"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.3/shippy-linux-arm64"
      sha256 "66e8b0d19fdfad627e8fc5c2e01dc6d24a2932456ca1ca22e47bb7dcb0b24d70"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.3/shippy-linux-amd64"
      sha256 "3d32b49cb9adf8fcd55a1f7694be2ef9ec75997f664f15549ba68931dd8d9fd2"
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
