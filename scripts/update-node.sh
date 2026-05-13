#!/usr/bin/env bash
#
# update-node.sh — run apt update/upgrade on a node, optionally reboot.
# Run as root.
#
set -euo pipefail

AUTO_REBOOT="${1:-false}"

export DEBIAN_FRONTEND=noninteractive
export NEEDRESTART_MODE=a
export NEEDRESTART_SUSPEND=1

echo "=== apt update ==="
apt-get update -y -o DPkg::Lock::Timeout=300

echo "=== apt upgrade ==="
apt-get upgrade -y \
  -o DPkg::Lock::Timeout=300 \
  -o DPkg::Options::="--force-confdef" \
  -o DPkg::Options::="--force-confold"

apt-get autoremove -y --purge || true

if [[ -f /var/run/reboot-required ]]; then
  echo "REBOOT-REQUIRED yes"
  cat /var/run/reboot-required.pkgs 2>/dev/null || true
  if [[ "$AUTO_REBOOT" == "true" ]]; then
    echo "auto-rebooting now"
    nohup bash -c 'sleep 2; systemctl reboot' >/dev/null 2>&1 &
  fi
else
  echo "REBOOT-REQUIRED no"
fi

echo "=== update-node OK ==="
