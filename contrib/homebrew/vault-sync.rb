class VaultSync < Formula
  desc "A fast, local-first note manager with multi-backend sync (Obsidian, Notion, Git)"
  homepage "https://vaultsync.dev"
  url "https://github.com/ishyverma/vault-sync.git",
      tag:      "v1.0.0",
      revision: "b37a0e58c958d101313027237cc77526e64c4c3c"
  license "MIT"

  depends_on "go" => :build

  def install
    ENV["CGO_ENABLED"] = "0"
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "./cmd/vault"
    system "go", "build", *std_go_args(ldflags: "-s -w -X main.version=#{version}"), "-o", bin/"vaultd", "./cmd/vaultd"
  end

  test do
    assert_match "vault", shell_output("#{bin}/vault --help")
  end
end
