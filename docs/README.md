# Installation & Setup

This document continues from the first three steps (clone/edit config/build) in the root `README.md`.

---

## Run

After building the executables into `bin/`, run each with `-config`:

    # Example: DNS (PowerDNS backend)
    ./bin/IBPDns -config config/dnsapi.json

    # Example: Monitor
    ./bin/IBPMonitor -config config/monitor.json

    # Example: Collator (optional)
    ./bin/IBPCollator -config config/collator.json

    # Example: Management API
    ./bin/mgmtApi -config config/mgmt.json

### Typical Order

1. **IBPMonitor**  
   - Performs periodic health checks (ping, SSL, WSS).
   - Publishes official results on `/results` (default port 6101).

2. **IBPDns**  
   - Receives queries from PowerDNS via HTTP (default port 6100).
   - Fetches official statuses from IBPMonitor to determine which members are online.
   - Optionally records usage stats to MySQL.

3. **mgmtApi** (optional)  
   - Provides REST endpoints for domain usage, events, membership info, etc.

4. **Bots** (optional)  
   - Discord or Matrix bots for chat-based monitoring or commands.

---

## Configuration Basics

- **System**  
  `WorkDir`, `LogLevel`, intervals, etc.

- **NATS**  
  `NodeID`, `Url`, `User`, `Pass`.

- **MySQL**  
  `Host`, `Port`, `User`, `Pass`, `DB`.

- **MaxMind**  
  `MaxmindDBPath`, `AccountID`, `LicenseKey` for geo lookups.

- **DNS / Monitor**  
  `ListenAddress`, `ListenPort`, etc.

- **Checks**  
  e.g., ping, ssl, wss, each with intervals.

See `config/*.json` for live examples.

---

## PowerDNS Integration

Configure PowerDNS to point to the DNS service (e.g. `http://127.0.0.1:6100/dns`):

    launch=remote
    remote-connection-string=http:url=http://127.0.0.1:6100/dns

Test with cURL:

    curl -X POST -H "Content-Type: application/json" \
         -d '{"method":"lookup","parameters":{"qname":"example.com","qtype":"A","remote":"1.2.3.4"}}' \
         http://127.0.0.1:6100/dns

---

## Testing

1. **Unit Tests**  
       go test ./...
2. **Integration**  
   - Start local MySQL and NATS.
   - Run `IBPMonitor`, then `IBPDns`.
   - `curl http://127.0.0.1:6101/results` to see official statuses.
3. **Logs**  
   - Check logs to confirm the DNS process is referencing the correct online members.
