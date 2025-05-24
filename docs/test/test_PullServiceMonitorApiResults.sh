#!/usr/bin/env bash
#
# Demo/test script to retrieve current official results from the serviceMonitor API
#
# Usage:
#   ./test_pull_results.sh [monitor_address] [monitor_port]
# 
# Default monitor_address: 127.0.0.1
# Default monitor_port: 6101

MONITOR_ADDR="${1:-127.0.0.1}"
MONITOR_PORT="${2:-6101}"

echo "Pulling official results from serviceMonitor at ${MONITOR_ADDR}:${MONITOR_PORT}..."
echo "-------------------------------------------------------"

# Attempt to use curl and print output
RESPONSE=$(curl -s "http://${MONITOR_ADDR}:${MONITOR_PORT}/results")

if [ -z "$RESPONSE" ]; then
  echo "No response or empty response received."
  exit 1
fi

echo "Official Results:"
echo "$RESPONSE"
echo "-------------------------------------------------------"
echo "Done."
