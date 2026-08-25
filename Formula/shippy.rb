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
  version "0.1.2"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.2/shippy-darwin-arm64"
      sha256 "513dcb153fc937753665a9d3547b1e3645b06a0840617633bb857161ba4a6357"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.2/shippy-darwin-amd64"
      sha256 "75785d41b4c09d8124bd04380e093ea25a711de8a7902a43ea27a2337caf7aef"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.2/shippy-linux-arm64"
      sha256 "71fac51c1eb36a38164611846976460ddb3be89976a4b2270bfa75b7372f8fea"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.1.2/shippy-linux-amd64"
      sha256 "7e654a9f6971ef703bb51a58de69c25a77abfb3f58b60e8a24b4cca7912a3578"
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
