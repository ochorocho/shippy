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
  version "0.2.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.2/shippy-darwin-arm64"
      sha256 "62b2558fad3023006d8d36b133561a4f1881cf45d1af985c18e407d9a7423ca0"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.2/shippy-darwin-amd64"
      sha256 "b434b5e91f44be8943dd9883ae1bfb3b784d670739876945d506037dc2afd307"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.2/shippy-linux-arm64"
      sha256 "813093f3191175e865a67278114bdc813567eb48935dc85d723a6c0e5b158660"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.2/shippy-linux-amd64"
      sha256 "a144eeee9457516ad240c33316a793dd4535c15f4f25db912250d918b3a31d05"
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
