#!/bin/sh
# VaultSync — one-line installer
# Usage: curl -sf https://vaultsync.dev/install | sh

set -eu

REPO="ishyverma/vault-sync"
VERSION="${1:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Linux)  os="Linux" ;;
    Darwin) os="Darwin" ;;
    *)      echo "Unsupported OS: $(uname -s)"; exit 1 ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="x86_64" ;;
    aarch64|arm64) arch="arm64" ;;
    *)            echo "Unsupported arch: $(uname -m)"; exit 1 ;;
  esac

  echo "${os}_${arch}"
}

fetch_release() {
  local os_arch="$1"
  
  if [ "$VERSION" = "latest" ]; then
    url="https://github.com/${REPO}/releases/latest/download/vault-sync_${os_arch}.tar.gz"
  else
    url="https://github.com/${REPO}/releases/download/${VERSION}/vault-sync_${os_arch}.tar.gz"
  fi

  tmpdir=$(mktemp -d)
  cd "$tmpdir"

  echo "  Downloading vault-sync for ${os_arch}..."
  
  if command -v curl >/dev/null 2>&1; then
    curl -sfL "$url" -o vault-sync.tar.gz
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O vault-sync.tar.gz
  else
    echo "Error: need curl or wget"
    exit 1
  fi

  tar xzf vault-sync.tar.gz
  
  # Also download the daemon binary
  daemon_url="$(echo "$url" | sed 's/vault-sync_/vault-sync_daemon_/')"
  if command -v curl >/dev/null 2>&1; then
    curl -sfL "$daemon_url" -o vaultd-sync.tar.gz 2>/dev/null || true
  else
    wget -q "$daemon_url" -O vaultd-sync.tar.gz 2>/dev/null || true
  fi
  if [ -f vaultd-sync.tar.gz ]; then
    tar xzf vaultd-sync.tar.gz 2>/dev/null || true
  fi

  echo "$tmpdir"
}

install_binaries() {
  local tmpdir="$1"

  echo "  Installing to ${INSTALL_DIR}..."

  mkdir -p "$INSTALL_DIR" 2>/dev/null || true

  if [ -f "$tmpdir/vault" ]; then
    install -m 755 "$tmpdir/vault" "${INSTALL_DIR}/vault"
    echo "    ${INSTALL_DIR}/vault"
  fi
  if [ -f "$tmpdir/vaultd" ]; then
    install -m 755 "$tmpdir/vaultd" "${INSTALL_DIR}/vaultd"
    echo "    ${INSTALL_DIR}/vaultd"
  fi

  rm -rf "$tmpdir"
}

main() {
  echo ""
  echo "  VaultSync Installer"
  echo "  ==================="
  echo ""

  os_arch=$(detect_platform)

  tmpdir=$(fetch_release "$os_arch")
  install_binaries "$tmpdir"

  echo ""
  echo "  ✓ VaultSync installed!"
  echo ""
  echo "  Next steps:"
  echo "    vault init         Set up your vault"
  echo "    vault --help       See available commands"
  echo "    vaultd start       Start the sync daemon"
  echo ""
}

main
