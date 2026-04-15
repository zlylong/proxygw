#!/bin/bash
# ProxyGW Installation & Deployment Script
# Supports both fresh installs and updates.
# Usage: ./install.sh

set -e

REPO_DIR="/root/proxygw"
export GO111MODULE=on

echo "=== ProxyGW Deployment Script ==="

echo "[1/6] Installing system dependencies..."
apt-get update
apt-get install -y build-essential nftables frr sqlite3 curl golang nodejs npm wget unzip

echo "[2/6] Setting up routing rules..."
if ! grep -q "100 tproxy" /etc/iproute2/rt_tables; then
    echo "100 tproxy" >> /etc/iproute2/rt_tables
fi
# Add rule if not exists
ip rule show | grep -q "fwmark 0x1 lookup tproxy" || ip rule add fwmark 1 table tproxy
ip route show table tproxy | grep -q "local default dev lo" || ip route add local default dev lo table tproxy

echo "[3/6] Setting up directory structure..."
mkdir -p "$REPO_DIR/config"
mkdir -p "$REPO_DIR/core/mosdns"
mkdir -p "$REPO_DIR/core/xray"
mkdir -p "$REPO_DIR/core/nftables"
mkdir -p "$REPO_DIR/systemd"

echo "[4/6] Building backend..."
cd "$REPO_DIR/backend"
if [ ! -f go.mod ]; then
    go mod init proxygw
fi
go mod tidy
go build -o proxygw-backend .

echo "[5/6] Creating Systemd service..."
cat << 'EOF' > "$REPO_DIR/systemd/proxygw.service"
[Unit]
Description=ProxyGW Backend Service
After=network.target network-online.target nss-lookup.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=/root/proxygw/backend
ExecStart=/root/proxygw/backend/proxygw-backend
Restart=on-failure
RestartSec=5
LimitNOFILE=1048576

# Security Sandboxing
NoNewPrivileges=yes
ProtectSystem=strict
PrivateTmp=yes
ProtectKernelTunables=yes
ProtectControlGroups=yes
RestrictSUIDSGID=yes
ReadWritePaths=-/root/proxygw -/usr/local/bin -/etc/frr

[Install]
WantedBy=multi-user.target
EOF
cp "$REPO_DIR/systemd/proxygw.service" /etc/systemd/system/

echo "[6/6] Starting services..."
systemctl daemon-reload
systemctl enable --now proxygw

# Automatically generate a secure password if it's a fresh install
# Wait a few seconds for the database to be initialized by the backend
sleep 3
if [ -f "$REPO_DIR/config/bootstrap_password.txt" ]; then
    echo "=========================================================="
    echo "ProxyGW is running! A random initial password was generated:"
    cat "$REPO_DIR/config/bootstrap_password.txt"
    echo "Please login at http://$(hostname -I | awk '{print $1}') and CHANGE IT immediately."
    echo "=========================================================="
else
    echo "=========================================================="
    echo "ProxyGW is running at http://$(hostname -I | awk '{print $1}')"
    echo "If this is a fresh install, check the backend logs for the bootstrap password."
    echo "=========================================================="
fi
