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

echo "[2/4] Building backend..."
cd "$REPO_DIR/backend"
export GO111MODULE=on
go mod tidy
go build -o proxygw-backend .

echo "[3/4] Updating Systemd services (if changed)..."
cp "$REPO_DIR/systemd/proxygw.service" /etc/systemd/system/ || true
cp "$REPO_DIR/systemd/mosdns.service" /etc/systemd/system/ || true
cp "$REPO_DIR/systemd/xray.service" /etc/systemd/system/ || true
systemctl daemon-reload

echo "[4/4] Restarting services..."
systemctl restart proxygw

echo "Update Complete!"