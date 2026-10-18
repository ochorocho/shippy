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
  version "0.2.6"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.6/shippy-darwin-arm64"
      sha256 "8f2082ffca8a78143f06046042e66bfd787a7bc5e0b8da1710325d40d2ae524a"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.6/shippy-darwin-amd64"
      sha256 "0da123ef47b572adf65f8b19c22748c160adf0c20f7720aca314729267b05ac4"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.6/shippy-linux-arm64"
      sha256 "ad3856b8d32f751c78e86c3c9dae9d2c0ed4d2e64a9f47bd3bf62ec812930011"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.6/shippy-linux-amd64"
      sha256 "d5be567dbe699c64c72639c96b0c6eb66ee39d12c89351d7e90678a2f84d71f3"
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
