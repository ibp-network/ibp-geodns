# ibp-geodns

## Overview

This software is a GeoDNS service built on top of PowerDNS. It dynamically manages DNS records for
a distributed network of members, ensuring high availability and optimal performance by directing clients
to the healthiest and geographically closest member services.

## Features

- **Dynamic DNS Management**: Automatically updates DNS records based on the real-time status of member services.
- **Health Monitoring**: Conducts various health checks (e.g., ping, SSL, WSS) to ensure service availability.
- **GeoIP Integration**: Uses MaxMind's GeoLite2 database to determine client locations and optimize DNS responses based on proximity.
- **ACME Challenge Support**: Dynamically handles ACME challenges for domain validation.
- **Continuous Configuration Updates**: Periodically fetches and updates member and service configurations from remote sources.
- **PowerDNS Integration**: Provides HTTP endpoints for PowerDNS, including DNS lookup and domain information endpoints.

## Installation

### Prerequisites

- Go 1.16+
- MaxMind GeoLite2-City.mmdb
- PowerDNS

### Steps

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/ibp-network/ibp-geodns.git
   cd ibp-geodns
   ```

2. **Build the Project**:
   ```sh
   go build -o geodns-service main.go
   ```

3. **Configure Environment**:
   Ensure the following environment variables or configuration files are set up correctly:
     ```sh
     export MEMBERS_URL=https://github.com/ibp-network/config/blob/main/members_professional.json
     export SERVICES_URL=https://github.com/ibp-network/config/blob/main/services_rpc.json
     export STATIC_ENTRIES_URL=https://github.com/ibp-network/config/blob/main/geodns-static.json
     export GEOIP_DB_PATH=GeoLite2-City.mmdb
     ```

## Usage

1. **Run the Service**:
   ```sh
   ./geodns-service
   ```

   This will start the GeoDNS service, initializing configurations, starting health checks, and setting up the HTTP server for PowerDNS integration.

2. **PowerDNS Integration**:
   Configure PowerDNS to use the GeoDNS service as its backend by pointing it to the HTTP server endpoint provided by the service:
   ```sh
   curl -X POST -H "Content-Type: application/json" -d '{"method": "lookup", "parameters": {"qname": "example.com", "qtype": "A", "remote": "1.2.3.4"}}' http://localhost:8080/dns
   ```

## Configuration

### Member Configuration (`members_professional.json`)

Define member nodes and their attributes including IP addresses, geographical locations, and service assignments.
The configuration file is located [here](https://github.com/ibp-network/config/blob/main/members_professional.json).

### Service Configuration (`services_rpc.json`)

Define services, their configurations, and provider endpoints. The configuration file is located [here](https://github.com/ibp-network/config/blob/main/services_rpc.json).

### Static Entries (`geodns-static.json`)

Define static DNS entries, including ACME challenges and other non-dynamic records.
The configuration file is located [here](https://github.com/ibp-network/config/blob/main/geodns-static.json).

## Health Checks

The service supports the following health checks:

- **Ping**: Verifies the availability of a member by pinging its IP address.
- **SSL**: Checks the validity and expiry of SSL certificates.
- **WSS**: Validates WebSocket Secure endpoints by sending and receiving JSON-RPC requests.

## Licensing

- **GeoLite2 Data**: The GeoLite2 data created by MaxMind is licensed under the Creative Commons Attribution-ShareAlike 4.0 International License (`CC-BY-SA-4.0-LICENSE`).
Download the GeoLite2 database from [MaxMind](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data/).
- **Project License**: The code for this project is licensed under the MIT License. See the `LICENSE` file for details.

## Acknowledgements

- [MaxMind GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
- [PowerDNS](https://www.powerdns.com/)

## Contact

For issues or feature requests, please create an issue on the [GitHub repository](https://github.com/ibp-network/ibp-geodns/issues).
