#!/usr/bin/env bash
#
# install-node.sh — idempotent provisioning for an IBP GeoDNS node in
# Tom's canonical layout (system-level systemd, `ibp` user, /opt/ibp-geodns/).
#
# Safe to re-run. Detects existing components and leaves them untouched.
#
# Usage: install-node.sh <deploy-pubkey-content> <node-id>
#
set -euo pipefail

DEPLOY_PUBKEY="${1:?usage: $0 <deploy-pubkey> <node-id>}"
NODE_ID="${2:?usage: $0 <deploy-pubkey> <node-id>}"

export DEBIAN_FRONTEND=noninteractive
export NEEDRESTART_MODE=a
export NEEDRESTART_SUSPEND=1

echo "=== 1/9 packages ==="
apt-get update -y -o DPkg::Lock::Timeout=300
apt-get install -y \
  -o DPkg::Lock::Timeout=300 \
  -o DPkg::Options::="--force-confdef" \
  -o DPkg::Options::="--force-confold" \
  qemu-guest-agent ufw curl jq ca-certificates python3 openssl \
  mysql-server pdns-server pdns-backend-remote
systemctl enable --now qemu-guest-agent || true

echo "=== 2/9 NATS server (skip if already provisioned) ==="
if [[ -f /etc/ibpdns/nats.conf ]]; then
  echo "  existing NATS config at /etc/ibpdns/nats.conf -- leaving alone"
else
  echo "  fresh install: dropping a local NATS server config"
  NATS_VER=2.11.12
  if ! command -v nats-server >/dev/null && [[ ! -x /opt/nats/nats-server ]]; then
    tmp=$(mktemp -d)
    curl -fsSL "https://github.com/nats-io/nats-server/releases/download/v${NATS_VER}/nats-server-v${NATS_VER}-linux-amd64.tar.gz" \
      | tar -xz -C "$tmp"
    install -d /opt/nats
    install -m 0755 "$tmp"/nats-server-v*/nats-server /opt/nats/
    rm -rf "$tmp"
  fi
  mkdir -p /etc/ibpdns /var/lib/nats
  cat > /etc/ibpdns/nats.conf <<'EOF'
server_name: "rotko-nats"
http: "0.0.0.0:8222"
max_payload: 8M
max_connections: 100
write_deadline: "6s"

# cluster routes get filled in by Tom / operator after first install
authorization {
  users = [
    { user: "NODE_ID_HERE", password: "CHANGE_ME" }
  ]
}
EOF
  cat > /etc/systemd/system/nats.service <<'EOF'
[Unit]
Description=NATS Messaging Server
After=network.target
[Service]
ExecStart=/opt/nats/nats-server -c /etc/ibpdns/nats.conf
Restart=on-failure
[Install]
WantedBy=multi-user.target
EOF
  systemctl daemon-reload
  systemctl enable nats
fi

echo "=== 3/9 MySQL: db + ibpdns user ==="
PFILE=/root/.ibpdns_mysql_pass
if [[ ! -s "$PFILE" ]]; then
  openssl rand -base64 18 > "$PFILE"
  chmod 600 "$PFILE"
fi
MP=$(cat "$PFILE")
mysql --protocol=socket -uroot <<EOF
CREATE DATABASE IF NOT EXISTS ibpdns CHARACTER SET utf8mb4;
CREATE USER IF NOT EXISTS 'ibpdns'@'localhost' IDENTIFIED BY '${MP}';
ALTER  USER             'ibpdns'@'localhost' IDENTIFIED BY '${MP}';
GRANT ALL ON ibpdns.* TO 'ibpdns'@'localhost';
FLUSH PRIVILEGES;
EOF

echo "=== 4/9 ibp user + canonical paths ==="
id ibp >/dev/null 2>&1 || useradd --system --user-group --create-home \
  --home-dir /home/ibp --shell /bin/bash ibp
# system users created earlier may have nologin shell + missing home — fix
usermod -d /home/ibp -s /bin/bash ibp || true
[[ -d /home/ibp ]] || install -d -o ibp -g ibp -m 0755 /home/ibp
install -d -o ibp -g ibp -m 0700 /home/ibp/.ssh
install -d -o ibp -g ibp -m 0755 \
  /opt/ibp-geodns /opt/ibp-geodns/bin /opt/ibp-geodns/.staging \
  /opt/ibp-geodns-monitor /opt/ibp-geodns-monitor/bin /opt/ibp-geodns-monitor/.staging \
  /opt/ibp-geodns-collator /opt/ibp-geodns-collator/bin /opt/ibp-geodns-collator/.staging
install -d -m 0755 /etc/ibpdns /var/lib/ibpdns /var/lib/ibpdns/maxmind
chown ibp:ibp /var/lib/ibpdns /var/lib/ibpdns/maxmind

echo "$DEPLOY_PUBKEY" > /home/ibp/.ssh/authorized_keys
chmod 600 /home/ibp/.ssh/authorized_keys
chown ibp:ibp /home/ibp/.ssh/authorized_keys

echo "=== 5/9 dns.json (only create if missing) ==="
if [[ ! -f /etc/ibpdns/dns.json ]]; then
  cat > /etc/ibpdns/dns.json <<EOF
{
  "System": {
    "WorkDir": "/opt/ibp-geodns/",
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
  "Maxmind": { "AccountID": "", "LicenseKey": "", "MaxmindDBPath": "/var/lib/ibpdns/maxmind/" },
  "Nats":    { "NodeID": "${NODE_ID}", "Url": "nats://127.0.0.1:4222", "User": "", "Pass": "" },
  "Mysql":   { "Host": "127.0.0.1", "Port": "3306", "User": "ibpdns", "Pass": "${MP}", "DB": "ibpdns" },
  "MonitorApi": { "ListenAddress": "127.0.0.1", "ListenPort": "6101" },
  "DnsApi":     { "ListenAddress": "0.0.0.0", "ListenPort": "6100",
                  "MonitorAddress": "", "MonitorPort": "",
                  "RefreshIntervalSeconds": 30 }
}
EOF
fi
# conservative reconcile: only fill in MISSING keys. Never overwrite values
# already present in an existing dns.json (Tom's may use different DB/user).
python3 - "$MP" "$NODE_ID" <<'PY'
import json, pathlib, sys
mp, node_id = sys.argv[1], sys.argv[2]
p = pathlib.Path('/etc/ibpdns/dns.json')
c = json.loads(p.read_text())
def ensure(section, key, val):
    s = c.setdefault(section, {})
    if not s.get(key):
        s[key] = val
ensure('Mysql', 'Host', '127.0.0.1')
ensure('Mysql', 'Port', '3306')
ensure('Mysql', 'User', 'ibpdns')
ensure('Mysql', 'DB',   'ibpdns')
ensure('Mysql', 'Pass', mp)
ensure('Maxmind', 'MaxmindDBPath', '/var/lib/ibpdns/maxmind/')
ensure('Nats',    'NodeID', node_id)
p.write_text(json.dumps(c, indent=2))
PY
chown ibp:ibp /etc/ibpdns/dns.json
chmod 640 /etc/ibpdns/dns.json

echo "=== 6/9 system-level ibpdns.service (only create if missing) ==="
if [[ ! -f /etc/systemd/system/ibpdns.service ]]; then
  cat > /etc/systemd/system/ibpdns.service <<'EOF'
[Unit]
Description=IBP GeoDNS
After=network.target nats.service mysql.service

[Service]
ExecStart=/opt/ibp-geodns/bin/ibp-dns -config /etc/ibpdns/dns.json
WorkingDirectory=/opt/ibp-geodns
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
systemctl enable ibpdns

echo "=== 7/9 sudoers for ibp user ==="
cat > /etc/sudoers.d/ibp-services <<'EOF'
# ibp manages its own services from deploy workflows
ibp ALL=(root) NOPASSWD: /bin/systemctl restart ibpdns, /bin/systemctl stop ibpdns, /bin/systemctl start ibpdns, /bin/systemctl is-active ibpdns, /bin/systemctl status ibpdns, /bin/systemctl restart ibpmonitor, /bin/systemctl stop ibpmonitor, /bin/systemctl start ibpmonitor, /bin/systemctl is-active ibpmonitor, /bin/systemctl restart ibpcollator, /bin/systemctl stop ibpcollator, /bin/systemctl start ibpcollator, /bin/systemctl is-active ibpcollator, /bin/systemctl restart pdns, /bin/systemctl is-active pdns, /bin/systemctl status pdns
EOF
chmod 0440 /etc/sudoers.d/ibp-services
visudo -cf /etc/sudoers.d/ibp-services >/dev/null

echo "=== 8/9 PowerDNS frontend (remote backend -> IBPDns on :6100) ==="
PDNS_API_KEY_FILE=/root/.pdns_api_key
[[ -s "$PDNS_API_KEY_FILE" ]] || { openssl rand -hex 24 > "$PDNS_API_KEY_FILE"; chmod 600 "$PDNS_API_KEY_FILE"; }
cat > /etc/powerdns/pdns.conf <<EOF
launch=remote
# IBPDns only handles POST / with the method in a JSON body, force JSON-over-POST
remote-connection-string=http:url=http://127.0.0.1:6100/dns,post=1,post_json=1,timeout=2000
remote-dnssec=yes
local-address=0.0.0.0,::
local-port=53
api=yes
api-key=$(cat "$PDNS_API_KEY_FILE")
webserver=yes
webserver-address=127.0.0.1
webserver-port=8081
disable-axfr=no
loglevel=4
EOF
chown root:pdns /etc/powerdns/pdns.conf
chmod 640 /etc/powerdns/pdns.conf
systemctl enable pdns

echo "=== 9/9 firewall (UFW, persists across reboots) ==="
ufw --force reset >/dev/null
ufw default deny incoming
ufw default allow outgoing

# public services
ufw allow 22/tcp                           comment 'ssh'
ufw allow 53/udp                           comment 'dns'
ufw allow 53/tcp                           comment 'dns/tcp'

# NATS cluster mesh — peer-to-peer between the IBP geodns nodes only.
# Allow whole prefixes so peers can rotate IPs within their range without
# us needing to update firewall rules every time.
NATS_PEER_V4=(
  192.96.202.0/24    # dns-01 (US, Leaseweb Manassas)
  91.90.166.0/24     # dns-02 (EU)
  181.174.168.0/24   # dns-04 (AR)
  160.22.180.0/23    # us (Rotko) — covers 160.22.180.0/24 and 160.22.181.0/24
)
NATS_PEER_V6=(
  2401:a860::/32     # us (Rotko) — our v6 prefix
  # add other operators' v6 ranges here as they come online
)
for net in "${NATS_PEER_V4[@]}"; do
  ufw allow from "$net" to any port 5222 proto tcp comment 'nats-cluster-peer'
done
for net in "${NATS_PEER_V6[@]}"; do
  ufw allow from "$net" to any port 5222 proto tcp comment 'nats-cluster-peer'
done

# DnsApi (6100) and MonitorApi (6101) speak to local processes only
# (PowerDNS -> IBPDns on 127.0.0.1, IBPDns -> monitor on 127.0.0.1).
# Lo traffic isn't filtered by UFW; explicit deny here makes sure no one
# can reach these from the public side even though the apps bind 0.0.0.0.
ufw deny 6100/tcp                          comment 'dns api - localhost only'
ufw deny 6101/tcp                          comment 'monitor api - localhost only'

ufw --force enable
# enable UFW persistence on reboot (Ubuntu default but be explicit)
systemctl enable ufw

# === retire legacy user-level geodns service if it was installed by an
# === earlier version of this script. Tom's system-level setup is canonical.
if id geodns >/dev/null 2>&1 && [[ -f /home/geodns/.config/systemd/user/ibpdns.service ]]; then
  echo "Retiring legacy user-level geodns ibpdns service"
  GU=$(id -u geodns)
  sudo -u geodns XDG_RUNTIME_DIR=/run/user/$GU systemctl --user stop ibpdns.service 2>/dev/null || true
  sudo -u geodns XDG_RUNTIME_DIR=/run/user/$GU systemctl --user disable ibpdns.service 2>/dev/null || true
  rm -f /etc/sudoers.d/geodns-pdns
fi
# also retire the original local nats-server.service if it was set up
if [[ -f /etc/systemd/system/nats-server.service ]] && [[ -f /etc/systemd/system/nats.service ]]; then
  systemctl stop nats-server.service 2>/dev/null || true
  systemctl disable nats-server.service 2>/dev/null || true
fi

echo
echo "=== install-node OK ==="
echo "  - deploy user:  ssh ibp@$(hostname -I | awk '{print $1}')"
echo "  - MySQL pass:   $PFILE"
echo "  - PDNS API key: $PDNS_API_KEY_FILE"
echo "  - Tom-aware: existing NATS/monitor/collator left untouched if present"
echo "  - Next: deploy.yaml ships ibp-dns binary to /opt/ibp-geodns/bin/"
