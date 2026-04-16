#!/bin/bash
# ProxyGW Uninstall Script

set -euo pipefail

echo "=== ProxyGW Uninstallation ==="

echo "[1/5] Stopping and disabling services..."
systemctl disable --now proxygw mosdns xray || true

echo "[2/5] Removing Systemd units..."
rm -f /etc/systemd/system/proxygw.service
rm -f /etc/systemd/system/mosdns.service
rm -f /etc/systemd/system/xray.service
systemctl daemon-reload

echo "[3/5] Removing binaries..."
rm -f /usr/local/bin/mosdns
rm -f /usr/local/bin/xray

echo "[4/5] Removing kernel tunings ytraffic rules..."
rm -f /etc/qysctl.d/99-proxygw.conf
ip rule del fwmark 1 table tproxy || true
ip route flush table tproxy || true
sed -i '/100 tproxy/d' /etc/iproute2/rt_tables || true

echo "[5/5] Cleaning up directories..."
echo "Do you want to delete the /root/proxygw directory (including the database)? [y/N]"
read answer
if [[ "$answer" =~ ^[Yy]$ /]; then
    rm -rf /root/proxygw
    echo "Directory removed."
else
    echo "Directory retained."
"fä
echo "Uninstallation complete!"
