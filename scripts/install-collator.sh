#!/usr/bin/env bash
#
# install-collator.sh — provisions IBP GeoDNS Collator on a node.
# Run as root. Idempotent.
#
# Binary itself is shipped by a deploy workflow living in
# `ibp-network/ibp-geodns-collator`.
#
# Usage: install-collator.sh <deploy-pubkey-content> <node-id>
#
set -euo pipefail

DEPLOY_PUBKEY="${1:?usage: $0 <deploy-pubkey> <node-id>}"
NODE_ID="${2:?usage: $0 <deploy-pubkey> <node-id>}"

id ibp >/dev/null 2>&1 || { echo "::error::run install-node.sh first to create the ibp user" >&2; exit 1; }

echo "=== collator dirs ==="
install -d -o ibp -g ibp -m 0755 \
  /opt/ibp-geodns-collator /opt/ibp-geodns-collator/bin /opt/ibp-geodns-collator/.staging \
  /opt/ibp-geodns-collator/tmp

echo "=== collator.json (only create if missing) ==="
if [[ ! -f /etc/ibpdns/collator.json ]]; then
  PFILE=/root/.ibpcollator_mysql_pass
  [[ -s "$PFILE" ]] || { openssl rand -base64 18 > "$PFILE"; chmod 600 "$PFILE"; }
  MP=$(cat "$PFILE")

  mysql --protocol=socket -uroot <<EOF
CREATE DATABASE IF NOT EXISTS collator CHARACTER SET utf8mb4;
CREATE USER IF NOT EXISTS 'collator'@'localhost' IDENTIFIED BY '${MP}';
ALTER  USER             'collator'@'localhost' IDENTIFIED BY '${MP}';
GRANT ALL ON collator.* TO 'collator'@'localhost';
FLUSH PRIVILEGES;
EOF

  cat > /etc/ibpdns/collator.json <<EOF
{
  "System": {
    "WorkDir": "/opt/ibp-geodns-collator/",
    "LogLevel": "Info",
    "ConfigUrls": {
      "StaticDNSConfig":       "https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/geodns-static.json",
      "MembersConfig":         "https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/members_professional.json",
      "ServicesConfig":        "https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/services_rpc.json",
      "IaasPricingConfig":     "https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/services_rpc_iaaspricing.json",
      "ServicesRequestsConfig":"https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/services_rpc_requests.json"
    },
    "ConfigReloadTime": 3600,
    "MinimumOfflineTime": 900
  },
  "Nats":    { "NodeID": "${NODE_ID}-COLLATOR", "Url": "nats://127.0.0.1:4222", "User": "", "Pass": "" },
  "Mysql":   { "Host": "127.0.0.1", "Port": "3306", "User": "collator", "Pass": "${MP}", "DB": "collator" },
  "Matrix":  { "HomeServerURL": "", "Username": "", "Password": "", "RoomID": "" },
  "CollatorApi": { "ListenAddress": "0.0.0.0", "ListenPort": "6102" }
}
EOF
fi
chown ibp:ibp /etc/ibpdns/collator.json
chmod 640 /etc/ibpdns/collator.json

echo "=== ibpcollator.service ==="
if [[ ! -f /etc/systemd/system/ibpcollator.service ]]; then
  cat > /etc/systemd/system/ibpcollator.service <<'EOF'
[Unit]
Description=IBP GeoDNS Collator
After=network.target nats.service mysql.service

[Service]
ExecStart=/opt/ibp-geodns-collator/bin/ibp-collator -config /etc/ibpdns/collator.json
WorkingDirectory=/opt/ibp-geodns-collator
User=ibp
Group=ibp
Restart=always
RestartSec=5
Environment=PATH=/usr/local/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF
fi
systemctl daemon-reload
systemctl enable ibpcollator

echo
echo "=== install-collator OK ==="
echo "  - dirs:    /opt/ibp-geodns-collator/"
echo "  - config:  /etc/ibpdns/collator.json"
echo "  - unit:    /etc/systemd/system/ibpcollator.service"
echo "  - next:    deploy a binary into /opt/ibp-geodns-collator/bin/ibp-collator"
echo "             (mirror this repo's deploy.yaml inside ibp-network/ibp-geodns-collator)"
