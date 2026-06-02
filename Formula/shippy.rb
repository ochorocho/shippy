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
      sha256 "095f081403e68e7eb07fc82c2304ff391cab7ec89e8a60c5ca50e795948a4f45"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-darwin-amd64"
      sha256 "308fe251ce5e84c5cd30906eb843678811827080ee89bd20d01e21d6a38df5af"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-linux-arm64"
      sha256 "636b9c51173d9a4057feb00fcee6bf6deb830c913dd1e46fb0d7166aa6d2378c"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.4/shippy-linux-amd64"
      sha256 "3bc18c95d4a62005cd2e8e8114d24b5121c2c832f182df41f4c78d5cbe555bb1"
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
