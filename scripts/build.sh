#!/bin/bash
set -euo pipefail

echo "=== Building ProxyGW Backend ==="
export GOPROXY=https://goproxy.cn,direct

cd /root/proxygw/backend
go build -o proxygw-backend

echo "Build successful. Restarting service..."
systemctl restart proxygw
echo "Service restarted."
