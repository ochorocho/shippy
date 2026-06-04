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
  version "0.0.9"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.9/shippy-darwin-arm64"
      sha256 "b85df52746f0d1fe6691b59f80b8a7ec8930afe2eb79a3d1eb3d49770b44b7ee"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.9/shippy-darwin-amd64"
      sha256 "2409a1c2b0f638a431998c74279f8fff81a85c765cd806d700aa3950ccba3aa8"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.9/shippy-linux-arm64"
      sha256 "110ae7f60d8708ae0113df258ccf6290296530d70d4ce55d20ea57b6b6e3b934"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.9/shippy-linux-amd64"
      sha256 "0f1c13995a85cc4bb50782a94f72d05f0712c077ac3549330a2dff32c3c690d5"
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
