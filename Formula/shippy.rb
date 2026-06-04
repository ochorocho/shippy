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
  version "0.0.8"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.8/shippy-darwin-arm64"
      sha256 "c764a4c52cfda0e7da02ccf0d28fac7cc69d2387b65c45c7b0d8604769b3c4b6"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.8/shippy-darwin-amd64"
      sha256 "e4b429f10c83f3834a886ed15e56ed1d501f8ef4e937eb73743c4c2d4f0c1254"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.8/shippy-linux-arm64"
      sha256 "bedf9ea8c23c08ff95fd6b32614c668d85b41128d1395b004e6939e86632c26d"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.0.8/shippy-linux-amd64"
      sha256 "4a53be2188414b8f9c71a6e8957f6665d90ac3465fe9f775c66038c1521aa6ec"
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
