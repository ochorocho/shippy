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
  version "0.2.1"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.1/shippy-darwin-arm64"
      sha256 "975c0b5227cc20f489f2892e74d9a449d602b2a0881ab01659491d6ca0724dfb"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.1/shippy-darwin-amd64"
      sha256 "487729ab71ddc0ea814c07f798667f632710c916cf4914cafdba49fe22f09f0f"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.1/shippy-linux-arm64"
      sha256 "7a864da62b42a0f65300b60d67a33783684d58198f0dc228501a54067c0e044f"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.1/shippy-linux-amd64"
      sha256 "982c3fb314492a2a01e902f1a33f596b584c9c9e6f13b009d567581c4fcaac11"
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
