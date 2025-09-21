# IBP GeoDNS Monitor

Health monitoring component for the IBP GeoDNS System v2 that performs distributed consensus-based health checks for RPC and ETH-RPC endpoints.

## Overview

The IBP GeoDNS Monitor performs periodic health checks on member infrastructure and participates in NATS-based consensus to determine official service status. It supports multiple check types with IPv4/IPv6 awareness and provides an API for retrieving offline members.

## Features

- **Multi-protocol health checks**: WebSocket (WSS), SSL certificate validation, ETH-RPC, and ping
- **Distributed consensus**: Multiple monitors vote on member status via NATS
- **IPv4/IPv6 dual-stack**: Independent health tracking for both protocols
- **Service-aware validation**: Type-specific checks (RPC vs ETHRPC)
- **Real-time API**: Query current offline members and their status

## Architecture

### Check Types

| Check | Type | Validates | Service Types |
|-------|------|-----------|---------------|
| `wss` | endpoint | WebSocket RPC connectivity, archive status, network match, peer count | RPC only |
| `ethrpc` | endpoint | ETH JSON-RPC methods, chain ID, syncing status | ETHRPC only |
| `ssl` | domain | SSL certificate validity (>5 days remaining) | RPC, ETHRPC |
| `ping` | site | ICMP connectivity and latency | All |

### Worker Pool System

- Configurable worker pool with staggered execution
- Priority queue based on check intervals
- Automatic retry and error recovery
- Per-check timeout configuration

## Configuration

### Local Configuration (`ibpmonitor.json`)

```json
{
  "System": {
    "WorkDir": "/path/to/workdir/",
    "LogLevel": "Debug",
    "ConfigUrls": {
      "StaticDNSConfig": "https://...",
      "MembersConfig": "https://...",
      "ServicesConfig": "https://..."
    },
    "ConfigReloadTime": 3600,
    "MinimumOfflineTime": 900
  },
  "Nats": {
    "NodeID": "MONITOR-NODE-1",
    "Url": "nats://server1:4222,nats://server2:4222",
    "User": "monitor-user",
    "Pass": "secure-password"
  },
  "MonitorApi": {
    "ListenAddress": "0.0.0.0",
    "ListenPort": "6101"
  },
  "CheckWorkers": {
    "numWorkers": 100,
    "separationInterval": 100
  },
  "Checks": [
    {
      "Name": "wss",
      "Enabled": 1,
      "CheckType": "endpoint",
      "Timeout": 90,
      "minimumInterval": 300,
      "ExtraOptions": {"ConnectTimeout": 10}
    }
  ]
}
```

### Check Configuration Options

**WSS Check**:
- `ConnectTimeout`: WebSocket connection timeout (seconds)

**ETHRPC Check**:
- `ConnectTimeout`: HTTP connection timeout (seconds)

**SSL Check**:
- `ConnectTimeout`: TLS handshake timeout (seconds)

**Ping Check**:
- `PingCount`: Number of ICMP packets
- `PingInterval`: Interval between packets (ms)
- `PingTimeout`: Total timeout (ms)
- `PingSize`: Packet size (bytes)
- `PingTTL`: Time-to-live value
- `MaxPacketLoss`: Maximum acceptable packet loss (%)
- `MaxLatency`: Maximum acceptable latency (ms)

## API Endpoints

### GET /results

Returns offline members grouped by check type. Only includes members that are currently offline according to the latest official consensus.

**Response Structure**:
```json
{
  "SiteResults": [
    {
      "CheckName": "ping",
      "IsIPv6": false,
      "Results": [
        {
          "MemberName": "member1",
          "ErrorText": "Connection timeout",
          "Data": {...},
          "IsIPv6": false,
          "Checktime": "2025-09-20T12:00:00Z"
        }
      ]
    }
  ],
  "DomainResults": [...],
  "EndpointResults": [...]
}
```

## Health Check Details

### WSS Check (RPC endpoints)
1. Establishes WebSocket connection with TLS verification
2. Validates chain network via `system_chain`
3. Verifies genesis state root (if configured)
4. Checks full archive status via `chain_getBlockHash(0)`
5. Monitors peer count and sync status via `system_health`
6. Requires: >5 peers, not syncing, correct network

### ETHRPC Check (Ethereum endpoints)
1. Connects via HTTPS with domain validation
2. Calls `eth_chainId` to verify network
3. Fetches current block via `eth_blockNumber`
4. Checks `net_version` for network validation
5. Ensures `eth_syncing` returns false
6. Validates network ID matches configuration

### SSL Check
1. Connects to port 443 with proper TLS handshake
2. Validates certificate chain
3. Checks certificate expiration (>5 days required)
4. Works for both RPC and ETHRPC services

## NATS Integration

The monitor participates in distributed consensus via NATS subjects:

- **Proposals**: `monitor.propose.*` - Submit check results
- **Voting**: `monitor.vote.*` - Vote on status changes  
- **Finalization**: `monitor.finalize.*` - Accept consensus results
- **State**: `monitor.stats.*` - Query downtime windows

Consensus requires:
- Minimum 2 votes
- Majority of active monitors
- Agreement on status change

## Building & Running

### Prerequisites
- Go 1.24.x or higher
- Access to NATS cluster
- Network access to monitored endpoints

### Build
```bash
go build -o ibp-geodns-monitor ./src/IBPMonitor.go
```

### Run
```bash
./ibp-geodns-monitor -config=/path/to/ibpmonitor.json
```

### Docker
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o monitor ./src/IBPMonitor.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/monitor /monitor
ENTRYPOINT ["/monitor"]
```

## Monitoring & Debugging

### Log Levels
- `Fatal`: Critical errors requiring immediate attention
- `Error`: Check failures and recoverable errors
- `Warn`: Configuration issues, degraded performance
- `Info`: Status changes, worker lifecycle
- `Debug`: Detailed check execution, consensus steps

### Key Metrics to Monitor
- Check execution rate per worker
- Consensus participation rate
- IPv4 vs IPv6 failure patterns
- Per-member offline duration
- Check type success rates

## Security Considerations

- TLS verification enforced for all HTTPS/WSS connections
- Domain names resolved to configured IPs (no DNS hijacking)
- NATS authentication required
- No sensitive data in API responses

## Dependencies

Core libraries:
- `github.com/gorilla/websocket` - WebSocket client
- `github.com/go-ping/ping` - ICMP implementation
- `github.com/ibp-network/ibp-geodns-libs` - Shared components

## License

See main repository LICENSE file.