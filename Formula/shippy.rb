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
  version "0.2.4"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.4/shippy-darwin-arm64"
      sha256 "09debccf0b6d54810e862a04782eec183fd6e494802653f07f13d77fb02cce51"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.4/shippy-darwin-amd64"
      sha256 "fc2ce6de8244112843307d9d91833f9b836850e0aec434100a4e03d51f42e5f3"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.4/shippy-linux-arm64"
      sha256 "a515b34f7e1b79d1439b21ab81eede75effce6d8d5a47be1d82641929a9dd407"
    end
    on_intel do
      url "https://github.com/ochorocho/shippy/releases/download/v0.2.4/shippy-linux-amd64"
      sha256 "f39bb2743b443da2feaf8a102fc8d0794efbbfd092190c79f268a9e984314db5"
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
