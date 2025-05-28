# IBP-GeoDNS Overview

## High-Level Components

1. **Monitors**  
   - Each node runs a `serviceMonitor` instance.
   - Performs periodic checks on members (ping, SSL, WSS).
   - Communicates with other Monitors over **NATS** to finalize official up/down status of members (consensus).
   - Exposes an HTTP endpoint (`/results`) for DNS APIs to retrieve official results.

2. **DNS APIs**  
   - Each node can run a `dnsApi` process that PowerDNS queries over HTTP (remote backend).
   - It combines:
     - Static DNS records (ACME challenges, etc.).
     - Dynamic records based on official Monitor results (which members are up/down).
   - Also tracks usage (domain requests, assigned member, etc.) for potential billing or analytics.

3. **Collator(s)**  
   - Optional: One or more specialized nodes that **aggregate** data from multiple sources:
     - **Monitor <-> Collator**: Collator can request downtime data or events from each Monitor.
     - **DNS Api <-> Collator**: Collator can request usage data from each DNS API node.
   - Collator then merges or aggregates the responses (e.g., for billing, reporting).

4. **MySQL / Stats**  
   - Each node can log usage or downtime events to a local MySQL (or shared DB).
   - Usage stats (DNS queries) are recorded by DNS API.
   - Downtime events are recorded by Monitors.

5. **Bots & Mgmt API**  
   - (Optional) Provide direct chat or REST-based commands to manage membership, usage queries, or other administrative tasks.

## Data Flow Example

1. **Health Checks** (Monitors)
   - Each Monitor checks local members (ping, SSL, WSS).
   - Proposes changes (if any) over NATS to other Monitors.
   - A majority vote finalizes official up/down status.

2. **DNS Queries** (DNS API)
   - PowerDNS calls `dnsApi` on port 6100.
   - `dnsApi` fetches official results from the local Monitor’s `/results` endpoint or from a consensus snapshot in memory.
   - If a domain is requested, it returns A/AAAA for the best/online member.

3. **Collator** (Aggregating Data)
   - The Collator connects to NATS.
   - When needed, it sends requests to:
     - **Monitors** (for downtime events, open or closed, between date ranges).
     - **DNS APIs** (for usage data about domains, members, or countries).
   - Collator collects these responses, merges them, and produces final reports (e.g., monthly billing).

