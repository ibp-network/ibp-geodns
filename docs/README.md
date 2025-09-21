# IBP GeoDNS

DNS API server and PowerDNS remote backend for the IBP GeoDNS System v2, providing geographic load balancing with health-aware routing.

## Overview

IBP GeoDNS is the core DNS component that integrates with PowerDNS to provide:
- Geographic routing based on MaxMind GeoIP data
- Dynamic A/AAAA responses with 30-second TTL
- Health-aware routing using official monitor consensus
- ACME challenge support for SSL certificates
- Usage accounting with MySQL persistence

## Features

- **PowerDNS Remote Backend**: Full implementation of PowerDNS JSON protocol
- **Geographic Load Balancing**: Routes users to nearest healthy infrastructure provider
- **IPv4/IPv6 Dual Stack**: Independent health tracking and routing for both protocols
- **Dynamic & Static Records**: Service records updated based on health; static NS/TXT from config
- **Usage Analytics**: Tracks queries by date, domain, member, country, ASN, and network
- **ACME Support**: Fetches TXT records via HTTP for SSL certificate validation

## Architecture

### PowerDNS Integration

The DNS API implements these PowerDNS remote backend methods:
- `initialize` - Backend initialization
- `lookup` - Dynamic DNS queries (A/AAAA with geo-routing)
- `list` - Static record enumeration
- `getDomainInfo` - Zone metadata
- `getAllDomains` - All zones served
- `getAllDomainMetadata` - Zone metadata
- `getDomainKeys` - DNSSEC keys
- `getMemberEvents` - Member outage history

### Routing Algorithm

1. Extract client IP and determine location via MaxMind
2. Query official monitor snapshot for member status
3. Filter members:
   - Skip if `Member.Override=true`
   - Skip if offline (per consensus)
   - Skip if missing required IP version
4. Calculate geographic distance to each healthy member
5. Return IP of closest member with 30s TTL

## Configuration

### Local Configuration (`ibpdns.json`)

```json
{
  "System": {
    "WorkDir": "/path/to/workdir/",
    "LogLevel": "Info",
    "ConfigUrls": {
      "StaticDNSConfig": "https://raw.githubusercontent.com/.../geodns-static.json",
      "MembersConfig": "https://raw.githubusercontent.com/.../members_professional.json",
      "ServicesConfig": "https://raw.githubusercontent.com/.../services_rpc.json"
    },
    "ConfigReloadTime": 3600,
    "MinimumOfflineTime": 900
  },
  "Nats": {
    "NodeID": "DNS-NODE-1",
    "Url": "nats://server1:4222,nats://server2:4222",
    "User": "dns-user",
    "Pass": "__SET_ME__"
  },
  "Mysql": {
    "Host": "localhost",
    "Port": "3306",
    "User": "ibpdns",
    "Pass": "__SET_ME__",
    "DB": "ibpdns"
  },
  "Maxmind": {
    "MaxmindDBPath": "/path/to/maxmind/",
    "AccountID": "your-account-id",
    "LicenseKey": "__SET_ME__"
  },
  "DnsApi": {
    "ListenAddress": "0.0.0.0",
    "ListenPort": "6100",
    "MonitorAddress": "127.0.0.1",
    "MonitorPort": "6101",
    "RefreshIntervalSeconds": 30
  }
}
```

### PowerDNS Configuration

```conf
# pdns.conf
launch=remote
remote-connection-string=http:url=http://localhost:6100/dns
```

## API Endpoints

### POST /dns
PowerDNS remote backend protocol. Handles all DNS queries and zone management.

**Request**:
```json
{
  "method": "lookup",
  "parameters": {
    "qname": "rpc.dotters.network",
    "qtype": "A",
    "remote": "203.0.113.45"
  }
}
```

**Response**:
```json
{
  "result": [
    {
      "qname": "rpc.dotters.network",
      "qtype": "A",
      "content": "198.51.100.10",
      "ttl": 30,
      "auth": true
    }
  ]
}
```

### GET /process?date=YYYY-MM-DD
Triggers manual usage flush to MySQL for specified date.

## DNS Record Types

### Dynamic Records
- **A/AAAA**: Geographic routing to nearest healthy member
- **TTL**: 30 seconds for dynamic responses

### Static Records
- **NS**: Nameserver records from remote config
- **TXT**: Static text records
- **SOA**: Serial uses current UTC timestamp

### Special Handling
- **ACME challenges**: `_acme-challenge.*` fetched via HTTP
  - Maximum 3 retry attempts
  - 512-byte response limit
  - 5-second timeout per attempt

## Usage Accounting

The system tracks DNS queries with these dimensions:
- **Temporal**: Date/time of query
- **Geographic**: Country, ASN, network (via MaxMind)
- **Service**: Domain queried, member selected
- **Protocol**: IPv4 vs IPv6

Data flow:
1. In-memory aggregation during operation
2. Automatic flush every 5 minutes
3. Manual flush via `/process` endpoint
4. MySQL persistence for reporting

## Monitor Integration

IBP GeoDNS polls the monitor API every 30 seconds to update its routing snapshot:
- Fetches from `http://monitor:6101/results`
- Updates official health status
- Routes only to members marked online

Health is tracked independently for IPv4 and IPv6, allowing partial availability.

## Building & Running

### Prerequisites
- Go 1.24.x or higher
- MySQL 5.7+
- NATS cluster access
- MaxMind GeoLite2 license

### Build
```bash
go build -o ibp-geodns ./src/IBPDns.go
```

### Run
```bash
./ibp-geodns -config=/path/to/ibpdns.json
```

### Docker
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o geodns ./src/IBPDns.go

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/geodns /geodns
ENTRYPOINT ["/geodns"]
```

## Database Schema

Key tables:
- `dns_usage`: Query statistics and aggregations
- `member_events`: Outage tracking and history
- `dns_cache`: Response caching (optional)

## Monitoring & Debugging

### Log Levels
- `Fatal`: Critical failures
- `Error`: Request failures, backend errors
- `Warn`: Configuration issues, fallback behavior
- `Info`: Startup, config reloads, routing decisions
- `Debug`: Detailed request/response, distance calculations

### Key Metrics
- Query rate by type (A/AAAA)
- Member selection distribution
- Geographic query origins
- Cache hit rates
- Response times

## Security Considerations

- Never expose credentials in configs (use `__SET_ME__` placeholders)
- TLS verification for remote config fetches
- ACME response size limits prevent abuse
- Member override mechanism for maintenance

## Dependencies

Core libraries:
- `github.com/ibp-network/ibp-geodns-libs` - Shared components
- `golang.org/x/net/publicsuffix` - TLD extraction
- PowerDNS 4.x - Authoritative DNS server

## License

See LICENSE file in repository root.