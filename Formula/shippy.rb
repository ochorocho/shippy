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
  version "0.0.7"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.7/shippy-darwin-arm64"
      sha256 "b539c855f020fa0b3676723950ae7486f203e03c1133f2a07dc0d1b859a925ff"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.7/shippy-darwin-amd64"
      sha256 "393057d7b5ba6654aba2692a56c7ffdbb100aeacb050125042f004572b692e14"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.7/shippy-linux-arm64"
      sha256 "ef0562272ee77ad525587f0a764d6229c69c1341ddaa7fe5c763d8e6c82451da"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.7/shippy-linux-amd64"
      sha256 "d8d98d2a687e8b141ef1d38f406d17e982b1c7a36c25442a511a1cb257d8e205"
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
