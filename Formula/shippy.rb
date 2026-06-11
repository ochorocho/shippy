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
  version "0.0.10"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.10/shippy-darwin-arm64"
      sha256 "e1f58ba4fa168b0a4887df611781df50bcf7919064a2a3cab23a888359b64d4f"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.10/shippy-darwin-amd64"
      sha256 "914fd3aadb3fbd80b2a6c5206306cfd7c2ea5823ec30d52df4bb21c8ad9e7039"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.10/shippy-linux-arm64"
      sha256 "a05283acb80e65e0ac66c19ab6421c84cd2e74310cfc8483e44a950798a4e445"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.10/shippy-linux-amd64"
      sha256 "1f45ff704706178f6b8b63ea11c69bf6639bf4b5cf6f3ad675310ac817bcd69e"
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
