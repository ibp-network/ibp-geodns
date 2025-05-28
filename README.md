# IBP DNS System

This repository provides a modular and extensible solution for dynamically managing DNS records, health checks, and load balancing across a decentralized collective of infrastructure providers.

The main documentation is in the `docs/` folder:
- `docs/README.md` (installation & setup instructions)
- `docs/README-components.md` (overview of major components)
- `docs/README-networking.md` (NATS-based architecture details)

---

## Quick Start

1. **Clone the repository**  
   git clone https://github.com/ibp-network/ibp-geodns.git  
   cd ibp-geodns

2. **Edit configuration**  
   Adjust `config/*.json` to match your environment (MySQL, NATS, etc.).

3. **Build**  
   For example:

       go build -o bin/IBPDns ./src/IBPDns/IBPDns.go
       go build -o bin/IBPMonitor ./src/IBPMonitor/IBPMonitor.go
       (Repeat similarly for mgmtApi, etc.)

4. **Next steps**  
   See `docs/README.md` for how to run and integrate with PowerDNS.
