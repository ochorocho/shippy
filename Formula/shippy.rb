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
  version "0.2.0"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-darwin-arm64"
      sha256 "17ade7c743bb184333d125c47bb65a5109d5b719c943b7a4b5d509668f1d5526"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-darwin-amd64"
      sha256 "9dc71c1bbe6620d729585aaa8dd1543c897be21c5183b3158053587bade9600a"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-linux-arm64"
      sha256 "aec276a24aa9332ec4df403bb35d51a186596ae35fd6bb4a6890542595aaa1e8"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.0/shippy-linux-amd64"
      sha256 "932b98ef2e105c814bdc1a48a6c5ae9c692163977859b145fc00b9f4c02124af"
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
