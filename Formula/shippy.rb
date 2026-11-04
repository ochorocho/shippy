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
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.7/shippy-darwin-arm64"
      sha256 "eaa2438c14673a44bd703b4a6f1cdd72ae1febf6092feaeef52b5c3f05c02c20"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.7/shippy-darwin-amd64"
      sha256 "24c54f23ace2291b906f5c322f7221debbd27743b1669417b8e025db6be79276"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.7/shippy-linux-arm64"
      sha256 "4ef629b602e097bb2860069f20c113387c11e3346c26af306602d11326a7485c"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.7/shippy-linux-amd64"
      sha256 "d9718a7b394a503faafcc7807cd4badb5751cd9b0a361046259d0156879d8ba9"
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
