# Overview of Components

1. **Monitor (IBPMonitor)**  
   - Periodically checks each member's site/domain/endpoint via ping, SSL, WSS, etc.
   - Publishes official status to a local `/results` endpoint and over NATS for consensus.

2. **DNS (IBPDns)**  
   - Receives PowerDNS queries over HTTP (remote backend).
   - Combines static records with dynamic A/AAAA records for members that are officially "online."
   - Records usage (optional) to MySQL or local stats.

3. **Collator (optional)**  
   - Aggregates usage or downtime events from multiple DNS and Monitor nodes via NATS.
   - Used for billing or reporting.

4. **Management API (optional)**  
   - Exposes REST endpoints for domain usage, member status, or admin tasks.
   - Could be integrated with Discord or Matrix bots for chat-based commands.

5. **Database (MySQL)**  
   - Stores usage stats, downtime events, etc.

---

## Typical Flow

1. **Monitor** checks members and finalizes official up/down via NATS.
2. **DNS** returns A/AAAA records only for members that are officially up.
3. **Collator** merges data from multiple nodes if needed.
