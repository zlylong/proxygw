package remote_deploy

import (
	"fmt"
	"strings"
)

func GenerateWGInstallScript(port int, serverPriv, clientPub, tunnelAddr string) string {
	script := `#!/bin/bash
set -e
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y wireguard iptables iproute2 curl
echo net.ipv4.ip_forward=1 > /etc/sysctl.d/99-wireguard-forward.conf
sysctl -p /etc/sysctl.d/99-wireguard-forward.conf
cat << 'WGCFG' > /etc/wireguard/wg0.conf
[Interface]
PrivateKey = %s
Address = %s
ListenPort = %d
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -A FORWARD -o wg0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT; iptables -t nat -A POSTROUTING -o $(ip route show default | awk '/default/ {print $5}' | head -1) -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -D FORWARD -o wg0 -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT; iptables -t nat -D POSTROUTING -o $(ip route show default | awk '/default/ {print $5}' | head -1) -j MASQUERADE

[Peer]
PublicKey = %s
AllowedIPs = %s
WGCFG
systemctl enable wg-quick@wg0
systemctl restart wg-quick@wg0
`

	clientIP := strings.Replace(tunnelAddr, ".1/24", ".2/32", 1)
	return fmt.Sprintf(script, serverPriv, tunnelAddr, port, clientPub, clientIP)
}

func GenerateVlessRealityInstallScript(port int, uuid, privateKey, shortId, serverName, dest string) string {
	script := `#!/bin/bash
set -e
export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install -y curl unzip
mkdir -p /usr/local/etc/xray
mkdir -p /usr/local/share/xray
curl -L -H "Cache-Control: no-cache" -o xray.zip https://github.com/XTLS/Xray-core/releases/latest/download/Xray-linux-64.zip
unzip -o xray.zip -d /usr/local/bin/xray-core
mv /usr/local/bin/xray-core/xray /usr/local/bin/xray
mv /usr/local/bin/xray-core/geoip.dat /usr/local/share/xray/
mv /usr/local/bin/xray-core/geosite.dat /usr/local/share/xray/
chmod +x /usr/local/bin/xray
rm -rf xray.zip /usr/local/bin/xray-core

cat << 'XCFG' > /usr/local/etc/xray/config.json
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [
    {
      "listen": "0.0.0.0",
      "port": %d,
      "protocol": "vless",
      "settings": {
        "clients": [
          {
            "id": "%s",
            "flow": "xtls-rprx-vision"
          }
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "%s",
          "xver": 0,
          "serverNames": [
            "%s"
          ],
          "privateKey": "%s",
          "shortIds": [
            "%s"
          ]
        }
      },
      "sniffing": {
        "enabled": true,
        "destOverride": ["http", "tls", "quic"]
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "tag": "direct"
    }
  ]
}
XCFG

cat << 'XSRV' > /etc/systemd/system/xray.service
[Unit]
Description=Xray Service
Documentation=https://github.com/xtls
After=network.target nss-lookup.target

[Service]
User=root
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_ADMIN CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ExecStart=/usr/local/bin/xray run -config /usr/local/etc/xray/config.json
Restart=on-failure
RestartPreventExitStatus=23
LimitNPROC=10000
LimitNOFILE=1000000

[Install]
WantedBy=multi-user.target
XSRV

systemctl daemon-reload
systemctl enable xray
systemctl restart xray
`

	return fmt.Sprintf(script, port, uuid, dest, serverName, privateKey, shortId)
}
