#!/bin/bash
echo "Installing dependencies..."
apt update && apt install -y build-essential  nftables frr sqlite3 curl golang nodejs npm

echo "Setting up routing..."
grep -q "100 tproxy" /etc/iproute2/rt_tables || echo "100 tproxy" >> /etc/iproute2/rt_tables
ip rule add fwmark 1 table tproxy
ip route add local default dev lo table tproxy

echo "Building backend..."
cd /root/proxygw/backend && go mod tidy && go build -o proxygw-backend

echo "Starting service..."
cp /root/proxygw/systemd/proxygw.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable proxygw --now
