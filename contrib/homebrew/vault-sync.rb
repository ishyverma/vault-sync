class VaultSync < Formula
  desc "Local-first note manager with multi-backend sync (Obsidian, Notion, Git)"
  homepage "https://vaultsync.dev"
  license "MIT"
  head "https://github.com/ishyverma/vault-sync.git", branch: "main"

  depends_on "go" => :build

  stable do
    url "https://github.com/ishyverma/vault-sync.git",
        tag:      "v1.0.0",
        revision: "3bb61560de1b374b36a69742da5c325fe5d5e0ad"
  end

  def install
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags, output: bin/"vault"), "./cmd/vault"
    system "go", "build", *std_go_args(ldflags: ldflags, output: bin/"vaultd"), "./cmd/vaultd"
  end

  def caveats
    <<~EOS
      To install the Vim/Neovim plugin, run:
        vault plugin install

      Or specify the target:
        vault plugin install --vim        # Vim
        vault plugin install --neovim     # Neovim

      This enables auto-sync on save (:w), :VaultSyncPush, and
      VaultSyncStatusline() for your status bar.

      To get started:
        vault init
        vault new my-first-note

      For background sync:
        vaultd start

      Full documentation: https://vaultsync.dev
    EOS
  end

  test do
    system bin/"vault", "--help"
    system bin/"vaultd", "--help"
  end
end
