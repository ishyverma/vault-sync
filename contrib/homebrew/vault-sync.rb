# typed: true
# frozen_string_literal: true

class VaultSync < Formula
  desc "A fast, local-first note manager with multi-backend sync (Obsidian, Notion, Git)"
  homepage "https://vaultsync.dev"
  license "MIT"
  version "1.0.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/ishyverma/vault-sync/releases/latest/download/vault-sync_Darwin_arm64.tar.gz"
      sha256 "TBD"
    else
      url "https://github.com/ishyverma/vault-sync/releases/latest/download/vault-sync_Darwin_x86_64.tar.gz"
      sha256 "TBD"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/ishyverma/vault-sync/releases/latest/download/vault-sync_Linux_arm64.tar.gz"
      sha256 "TBD"
    else
      url "https://github.com/ishyverma/vault-sync/releases/latest/download/vault-sync_Linux_x86_64.tar.gz"
      sha256 "TBD"
    end
  end

  def install
    bin.install "vault"
    bin.install "vaultd"
  end

  test do
    assert_match "vault", shell_output("#{bin}/vault --help")
  end
end
