#!/usr/bin/env bash
set -euo pipefail

VAULTD_BIN="${VAULTD_BIN:-vaultd}"
LAUNCHD_LABEL="dev.vaultsync.vaultd"

detect_platform() {
  case "$(uname -s)" in
    Darwin*) echo "darwin" ;;
    Linux*)  echo "linux" ;;
    *)       echo "unsupported" ;;
  esac
}

install_launchd() {
  local plist="$HOME/Library/LaunchAgents/$LAUNCHD_LABEL.plist"
  mkdir -p "$(dirname "$plist")"

  cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>$LAUNCHD_LABEL</string>
  <key>ProgramArguments</key>
  <array>
    <string>$(which "$VAULTD_BIN" 2>/dev/null || echo "/usr/local/bin/$VAULTD_BIN")</string>
    <string>start</string>
  </array>
  <key>KeepAlive</key>
  <true/>
  <key>RunAtLoad</key>
  <true/>
  <key>StandardOutPath</key>
  <string>$HOME/Library/Logs/vaultd.log</string>
  <key>StandardErrorPath</key>
  <string>$HOME/Library/Logs/vaultd.log</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>/usr/local/bin:/usr/bin:/bin</string>
  </dict>
</dict>
</plist>
EOF

  launchctl unload "$plist" 2>/dev/null || true
  launchctl load "$plist"
  echo "✓ vaultd launchd service installed (label: $LAUNCHD_LABEL)"
}

install_systemd() {
  local service_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
  mkdir -p "$service_dir"

  local service_file="$service_dir/vaultsync-vaultd.service"
  cat > "$service_file" <<EOF
[Unit]
Description=VaultSync Daemon
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=$(which "$VAULTD_BIN" 2>/dev/null || echo "/usr/local/bin/$VAULTD_BIN") start
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
EOF

  systemctl --user daemon-reload
  systemctl --user enable vaultsync-vaultd.service
  systemctl --user start vaultsync-vaultd.service
  echo "✓ vaultd systemd user service installed"
}

main() {
  case "$(detect_platform)" in
    darwin) install_launchd ;;
    linux)  install_systemd ;;
    *)      echo "Unsupported platform. Only macOS and Linux are supported."; exit 1 ;;
  esac
}

main "$@"
