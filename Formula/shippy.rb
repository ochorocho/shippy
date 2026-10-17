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
  version "0.2.5"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.5/shippy-darwin-arm64"
      sha256 "f44fcdfdc8619b1f9446119315d9197645bd7d786b2e5e6bcf5f862fb7910dd4"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.5/shippy-darwin-amd64"
      sha256 "151c2aa5e309c3712d580838500c8c7c5c6e500017f87bc18aa3c9f28c2c0501"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.5/shippy-linux-arm64"
      sha256 "67a3a638df2251c06d8edb82e796de1a1c306c67f900e452b66a293fabc07a3e"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.5/shippy-linux-amd64"
      sha256 "5901e6b1e4fa7d9853812088cfa65387ae45f970afc687a1e91d8c48d5160f99"
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
