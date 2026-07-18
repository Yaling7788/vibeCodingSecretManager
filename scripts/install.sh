#!/bin/sh
set -eu

REPO_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
BIN_DIR=${VCSM_BIN_DIR:-"$HOME/.local/bin"}

if ! command -v go >/dev/null 2>&1; then
  echo "Go 1.24 or newer is required to build VCSM." >&2
  exit 1
fi

mkdir -p "$BIN_DIR"
cd "$REPO_DIR"
go build -trimpath -o "$BIN_DIR/vcsm" ./cmd/vcsm
go build -trimpath -o "$BIN_DIR/vcsm-broker" ./cmd/vcsm-broker
chmod 700 "$BIN_DIR/vcsm" "$BIN_DIR/vcsm-broker"

case "$(uname -s)" in
  Darwin)
    service_dir="$HOME/Library/LaunchAgents"
    service_file="$service_dir/dev.vcsm.broker.plist"
    mkdir -p "$service_dir"
    sed "s|__BROKER__|$BIN_DIR/vcsm-broker|g" "$REPO_DIR/scripts/service/dev.vcsm.broker.plist" > "$service_file"
    chmod 600 "$service_file"
    launchctl bootout "gui/$(id -u)/dev.vcsm.broker" >/dev/null 2>&1 || true
    launchctl bootstrap "gui/$(id -u)" "$service_file"
    ;;
  Linux)
    service_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
    service_file="$service_dir/vcsm-broker.service"
    mkdir -p "$service_dir"
    sed "s|__BROKER__|$BIN_DIR/vcsm-broker|g" "$REPO_DIR/scripts/service/vcsm-broker.service" > "$service_file"
    chmod 600 "$service_file"
    systemctl --user daemon-reload
    systemctl --user enable --now vcsm-broker.service
    ;;
  *)
    echo "Use scripts/install.ps1 on Windows." >&2
    exit 1
    ;;
esac

echo "Installed vcsm and its per-user broker."
echo "Ensure $BIN_DIR is on PATH, then run: vcsm init"
