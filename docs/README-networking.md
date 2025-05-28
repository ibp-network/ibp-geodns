# NATS Networking

This system uses [NATS](https://nats.io/) to coordinate:

1. **Monitor <-> Monitor**  
   - **Purpose**: Official up/down consensus (voting).
   - **Subjects**:
     - `consensus.propose`: A Monitor proposes a change in member status (online/offline).
     - `consensus.vote`: Peers vote yes/no based on their own local results.
     - `consensus.finalize`: Announces the final official status after majority is reached.

2. **Monitor <-> Collator**  
   - **Purpose**: Collator requests downtime data from multiple Monitors.
   - **Possible Subjects** (suggested):
     - `monitor.stats.getDowntime`: Collator sends a request specifying **start date**, **end date**, and optional **member** filter.
     - `monitor.stats.downtimeData`: Each Monitor responds with relevant downtime events (open or closed) in that date range.

   **Flow** example:
   - Collator publishes a `monitor.stats.getDowntime` message with:
   
         {
           "startTime": "2024-01-01T00:00:00Z",
           "endTime": "2024-01-31T23:59:59Z",
           "memberName": "optionalOrBlankForAll"
         }
   
   - Each Monitor listens for that subject, queries local data (MySQL or in-memory events), and replies on `monitor.stats.downtimeData` with:
   
         {
           "nodeID": "MONITOR-1",
           "events": [
             {"memberName":"Stkdio","checkType":"site", "startTime":"...", "endTime":"...", "status": false, ...},
             ...
           ]
         }
   
   - Collator collects all replies, merges them, and produces a consolidated downtime report.

3. **DNS (IBPDns) <-> Collator**  
   - **Purpose**: Collator requests usage data from DNS nodes to handle billing or usage analytics.
   - **Possible Subjects** (suggested):
     - `dns.usage.getUsage`: Collator requests usage stats for a domain(s) or entire date range from each DNS node.
     - `dns.usage.usageData`: DNS node replies with aggregated or raw usage data.

   **Flow** example:
   - Collator publishes a `dns.usage.getUsage` message with:
   
         {
           "startDate":"2024-01-01",
           "endDate":"2024-01-31",
           "domain":"optionalOrBlankForAll",
           "memberName":"optional",
           "country":"optional"
         }
   
   - Each DNS node listens for that subject, runs queries in local stats or MySQL, responds on `dns.usage.usageData`, for example:
   
         {
           "nodeID":"DNS-02",
           "usageRecords":[
             {"date":"2024-01-10","domain":"foo.com","memberName":"Stakeplus","countryCode":"US","hits":147},
             ...
           ]
         }
   
   - Collator merges usage records from all DNS nodes, generating final usage totals for each domain, member, country, etc.

---

## Example Lifecycle

### 1. Monitor <-> Monitor: Finalizing Member Status
1. A Monitor sees a member's ping check fail.
2. That Monitor sends a `consensus.propose` (subject: `consensus.propose`) with `[memberName, checkType, proposedStatus=false]`.
3. Other Monitors see the proposal, compare with local data, respond with `consensus.vote`.
4. After majority is reached, a final `consensus.finalize` is broadcast. The member is “officially offline.”

### 2. Monitor <-> Collator: Requesting Downtime
1. Collator publishes `monitor.stats.getDowntime`, requesting events for January.
2. Each Monitor queries its local DB for downtime events in that window, replies with `monitor.stats.downtimeData`.
3. Collator merges the replies into a comprehensive downtime table.

### 3. DNS (IBPDns) <-> Collator: Requesting Usage
1. Collator publishes `dns.usage.getUsage`, specifying `domain=foo.com` for January.
2. Each DNS node replies with usage data on `dns.usage.usageData`.
3. Collator merges usage for `foo.com` across the entire cluster.

---

## Subject Summary

| Subject                          | Producer        | Consumer        | Purpose                                               |
|----------------------------------|-----------------|-----------------|-------------------------------------------------------|
| **consensus.propose**           | Monitor         | Monitors        | Propose member status changes (online/offline)       |
| **consensus.vote**              | Monitor         | Monitors        | Votes for/against a proposed status                  |
| **consensus.finalize**          | Monitor         | Monitors        | Announces final official result                      |
| **monitor.stats.getDowntime**   | Collator        | Monitors        | Request downtime events for a date range/member(s)   |
| **monitor.stats.downtimeData**  | Monitors        | Collator        | Return downtime events from local DB                |
| **dns.usage.getUsage**          | Collator        | DNS (IBPDns)    | Request usage stats for domain(s)/member(s)/range    |
| **dns.usage.usageData**         | DNS (IBPDns)    | Collator        | Return usage records from local stats or DB          |

---

## Implementation Notes

- **Request/Reply** pattern:
  - Collator can use a unique `ReplyTo` subject or ephemeral inbox so each Monitor or DNS node knows where to respond.
  - Alternatively, we could define dedicated “request” and “response” subjects if we prefer a broadcast approach with each node replying individually.

- **Payload**:
  - Typically JSON, with fields for filters (`startDate`, `endDate`, `domain`, `member`, etc.).
  - Each node includes `nodeID` in the response to identify the source.
