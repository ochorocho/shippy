# Homebrew formula for shippy (prebuilt-binary install).
#
# Install via tap:
#   brew tap ochorocho/shippy https://github.com/ochorocho/shippy
#   brew trust ochorocho/shippy
#   brew install shippy
#
# Do NOT hand-edit the url/sha256 values below — run `make brew-formula`
# (or scripts/update-formula.sh) to bump them for a release; the Release
# workflow does this automatically on tagged builds.
class Shippy < Formula
  desc "Zero-downtime deployment tool for Composer based PHP projects"
  homepage "https://github.com/ochorocho/shippy"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.8/shippy-darwin-arm64"
      sha256 "b8033bdd04d55ca69d412d7694eae74da9edb3403f6dc8e37564fbfabeca057e"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.8/shippy-darwin-amd64"
      sha256 "ac55f4ad0a4206e7e8da35ff1712d89ddaa16e4a6e5de8383a7c9814b0ac2d87"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.8/shippy-linux-arm64"
      sha256 "541a5f713cee7fb26a58a8200f338ef1e31880b9132027d914bc2dc3a7572daf"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.8/shippy-linux-amd64"
      sha256 "aa52789337e176d3da46b0581cfe20373836195516c47fd18dc4aea69bb0c7c8"
    end
  end

  def install
    # Exactly one prebuilt binary is staged for the host platform; rename to `shippy`.
    binary = Dir["*"].find { |f| File.file?(f) }
    bin.install binary => "shippy"
  end

  test do
    assert_match "shippy version #{version}", shell_output("#{bin}/shippy version")
    assert_match "Usage:", shell_output("#{bin}/shippy --help")

    # Offline workflow: scaffold a config, validate it, preview a deploy.
    (testpath/"composer.json").write '{"name":"acme/site"}'
    system bin/"shippy", "init"
    assert_path_exists testpath/".shippy.yaml"
    system bin/"shippy", "config", "validate"
    assert_match "composer.json", shell_output("#{bin}/shippy deploy production --dry-run --verbose")
  end
end
