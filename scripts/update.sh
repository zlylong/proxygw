#!/bin/bash
# ProxyGW Update Script
# Executes a full sync and rebuild from the remote repository.

set -euo pipefail

REPO_DIR="/root/proxygw"
cd "$REPO_DIR"

echo "=== ProxyGW Update ==="
echo "[1/4] Pulling latest changes..."
git fetch origin
git reset --hard origin/main

echo "[2/4] Downloading backend from GitHub Releases..."
ARCH=$(uname -m)
PROXYGW_LATEST=$(curl -s -4 https://api.github.com/repos/zlylong/proxygw/releases/latest | grep "tag_name": | sed -E "s/.*\"([^\"]+)\".*/\1/")
if [ -z "$PROXYGW_LATEST" ]; then
    echo "Error: Failed to fetch ProxyGW latest version!"
    exit 1
fi
if [ "$ARCH" = "x86_64" ]; then
    wget -q -4 -O "$REPO_DIR/backend/proxygw-backend" "https://github.com/zlylong/proxygw/releases/download/${PROXYGW_LATEST}/proxygw-backend-linux-amd64"
elif [ "$ARCH" = "aarch64" ]; then
    wget -q -4 -O "$REPO_DIR/backend/proxygw-backend" "https://github.com/zlylong/proxygw/releases/download/${PROXYGW_LATEST}/proxygw-backend-linux-arm64"
fi
chmod +x "$REPO_DIR/backend/proxygw-backend"

echo "[3/4] Updating Systemd services (if changed)..."
cp "$REPO_DIR/systemd/proxygw.service" /etc/systemd/system/ || true
cp "$REPO_DIR/systemd/mosdns.service" /etc/systemd/system/ || true
cp "$REPO_DIR/systemd/xray.service" /etc/systemd/system/ || true
systemctl daemon-reload

echo "[4/4] Restarting services..."
systemctl restart proxygw

echo "Update Complete!"