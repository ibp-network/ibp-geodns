# IBP-GeoDNS

Welcome to IBP-GeoDNS, a modular and extensible solution for dynamically managing DNS records, health checks, and load balancing across a decentralized collective of infrastructure providers.

All **complete documentation** is located in the [`docs/`](./docs) directory.

Please see:
- [`docs/README.md`](./docs/README.md) for an overview of the documentation.
- [`docs/INSTALLATION_AND_SETUP.md`](./docs/INSTALLATION_AND_SETUP.md) for detailed installation/build/run steps.

---

## Quick Start (First 3 Steps)

1. **Clone Repository:**

    git clone https://github.com/ibp-network/ibp-geodns.git  
    cd ibp-geodns

2. **Edit Configuration:**

   - The `config/*.json` files contain MySQL, NATS, MaxMind, etc.
   - Adjust to match your environment.

3. **Build:**

    go build -o bin/dnsApi ./src/dnsApi/dnsApi.go

   (Repeat similarly for `serviceMonitor`, `mgmtApi`, etc.)

For the **remaining** steps (run, usage, integration with PowerDNS, etc.), go to
[`docs/INSTALLATION_AND_SETUP.md`](./docs/INSTALLATION_AND_SETUP.md).
