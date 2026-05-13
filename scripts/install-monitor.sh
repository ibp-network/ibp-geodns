#!/usr/bin/env bash
#
# install-monitor.sh — provisions IBP GeoDNS Monitor on a node.
# Run as root. Idempotent — safe to re-run; leaves existing files alone.
#
# This script provisions the *infrastructure* for the monitor service.
# The binary itself is shipped by a deploy workflow living in
# `ibp-network/ibp-geodns-monitor` (mirror of ibp-geodns/deploy.yaml).
#
# Usage: install-monitor.sh <deploy-pubkey-content> <node-id>
#
set -euo pipefail

DEPLOY_PUBKEY="${1:?usage: $0 <deploy-pubkey> <node-id>}"
NODE_ID="${2:?usage: $0 <deploy-pubkey> <node-id>}"

# Reuse install-node.sh's ibp user + sudoers setup; this script only adds the
# monitor-specific bits.
id ibp >/dev/null 2>&1 || { echo "::error::run install-node.sh first to create the ibp user" >&2; exit 1; }

echo "=== monitor dirs ==="
install -d -o ibp -g ibp -m 0755 \
  /opt/ibp-geodns-monitor /opt/ibp-geodns-monitor/bin /opt/ibp-geodns-monitor/.staging \
  /opt/ibp-geodns-monitor/tmp /opt/ibp-geodns-monitor/tmp/maxmind

echo "=== monitor.json (only create if missing) ==="
if [[ ! -f /etc/ibpdns/monitor.json ]]; then
  PFILE=/root/.ibpmonitor_mysql_pass
  [[ -s "$PFILE" ]] || { openssl rand -base64 18 > "$PFILE"; chmod 600 "$PFILE"; }
  MP=$(cat "$PFILE")

  mysql --protocol=socket -uroot <<EOF
CREATE DATABASE IF NOT EXISTS ibpmonitor CHARACTER SET utf8mb4;
CREATE USER IF NOT EXISTS 'ibp'@'localhost' IDENTIFIED BY '${MP}';
ALTER  USER             'ibp'@'localhost' IDENTIFIED BY '${MP}';
GRANT ALL ON ibpmonitor.* TO 'ibp'@'localhost';
FLUSH PRIVILEGES;
EOF

  cat > /etc/ibpdns/monitor.json <<EOF
{
  "System": {
    "WorkDir": "/opt/ibp-geodns-monitor/",
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
  "Nats":    { "NodeID": "${NODE_ID}", "Url": "nats://127.0.0.1:4222", "User": "", "Pass": "" },
  "Mysql":   { "Host": "127.0.0.1", "Port": "3306", "User": "ibp", "Pass": "${MP}", "DB": "ibpmonitor" },
  "Maxmind": { "MaxmindDBPath": "/opt/ibp-geodns-monitor/tmp/maxmind/", "AccountID": "", "LicenseKey": "" },
  "MonitorApi": { "ListenAddress": "0.0.0.0", "ListenPort": "6101" },
  "CheckWorkers": { "Concurrency": 4, "WorkerSeparationMs": 200 },
  "Checks": { }
}
EOF
fi
chown ibp:ibp /etc/ibpdns/monitor.json
chmod 640 /etc/ibpdns/monitor.json

echo "=== ibpmonitor.service ==="
if [[ ! -f /etc/systemd/system/ibpmonitor.service ]]; then
  cat > /etc/systemd/system/ibpmonitor.service <<'EOF'
[Unit]
Description=IBP GeoDNS Monitor
After=network.target nats.service mysql.service

[Service]
ExecStart=/opt/ibp-geodns-monitor/bin/ibp-monitor -config /etc/ibpdns/monitor.json
WorkingDirectory=/opt/ibp-geodns-monitor
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
systemctl enable ibpmonitor

echo
echo "=== install-monitor OK ==="
echo "  - dirs:    /opt/ibp-geodns-monitor/"
echo "  - config:  /etc/ibpdns/monitor.json"
echo "  - unit:    /etc/systemd/system/ibpmonitor.service"
echo "  - next:    deploy a binary into /opt/ibp-geodns-monitor/bin/ibp-monitor"
echo "             (mirror this repo's deploy.yaml inside ibp-network/ibp-geodns-monitor)"
