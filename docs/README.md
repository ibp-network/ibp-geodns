# IBP-GeoDNS: Installation & Setup

This file **continues** from the first three steps (clone/edit config/build) mentioned in the root `README.md`.

---

## 4. Run

After building the executables into `bin/`, run each with `-config`:

    # Example: DNS API (PowerDNS backend)
    ./bin/dnsApi -config config/dnsapi.json

    # Example: Service Monitor
    ./bin/serviceMonitor -config config/monitor.json

    # Example: Management API
    ./bin/mgmtApi -config config/config.json

### Typical Order

1. **serviceMonitor**:  
   - Periodic health checks on members (ping/ssl/wss).  
   - Publishes official results on `/results` (port 6101).

2. **dnsApi**:  
   - Queries serviceMonitor, answers DNS queries from PowerDNS via HTTP (port 6100).

3. **mgmtApi** (optional):  
   - Provides domain usage, events, or membership info on port 6110.

4. **Bots** (optional):  
   - Discord or Matrix for chat-based management commands.

---

## Configuration Basics

- **System**  
  - `WorkDir`, `LogLevel`, intervals (ConfigReloadTime, CacheSaveTime, etc.).  
  - `ConfigUrls` (static DNS, members, services).

- **Nats**  
  - `NodeID`, `Url`, `User`, `Pass`.

- **Mysql**  
  - `Host`, `Port`, `User`, `Pass`, `DB`.

- **Maxmind**  
  - `MaxmindDBPath`, `AccountID`, `LicenseKey`.

- **MonitorApi / DnsApi / MgmtApi**  
  - `ListenAddress`, `ListenPort`, optional auth keys.

- **Checks**  
  - e.g. ping, ssl, wss checks with intervals.

See `config/*.json` for real examples.

---

## Port & Connection Summary

| Service          | Default Port | Purpose                                     |
|------------------|-------------|---------------------------------------------|
| dnsApi           | 6100        | PowerDNS remote-backend HTTP                |
| serviceMonitor   | 6101        | Health checks, official results endpoint    |
| mgmtApi          | 6110        | Management REST (usage/events/billing)      |
| NATS             | 4222        | Pub-sub for consensus across IBP-GeoDNS     |

---

## PowerDNS Integration

Configure PowerDNS to call the DNS API endpoint (for example, `http://127.0.0.1:6100/dns`):

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
   - Run `serviceMonitor`, then `dnsApi`.
   - Check logs, or cURL `/results` from `serviceMonitor` on port 6101.

3. **Monitor Logs**
   - If `dnsApi` sees official statuses from `serviceMonitor`, it should return the correct IP addresses.

---

## More Info

- [**README.md**](../README.md)  
- [**docs/README.md**](./README.md)  
- [**ibp-geodns.md**](./ibp-geodns.md) for architecture details.  
- [**ibp-geodns-networking.md**](./ibp-geodns-networking.md) for NATS-based pub-sub.

