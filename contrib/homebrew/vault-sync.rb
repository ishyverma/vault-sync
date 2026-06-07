class VaultSync < Formula
  desc "A fast, local-first note manager with multi-backend sync (Obsidian, Notion, Git)"
  homepage "https://vaultsync.dev"
  license "MIT"

  depends_on "go" => :build

  stable do
    url "https://github.com/ishyverma/vault-sync.git",
        tag:      "v1.0.0",
        revision: "3bb6156d1c3bd6a5c41214500bb9448cf8eb8c87"
  end

  head do
    url "https://github.com/ishyverma/vault-sync.git", branch: "main"
  end

  def install
    ENV["CGO_ENABLED"] = "0"
    ldflags = "-s -w -X main.version=#{version}"
    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/vault"
    system "go", "build", *std_go_args(ldflags: ldflags), "-o", bin/"vaultd", "./cmd/vaultd"
  end

  test do
    assert_match "vault", shell_output("#{bin}/vault --help")
  end
end
