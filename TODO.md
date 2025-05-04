## IBP-GeoDNS

- Figure out a way to allow members to receive push notifications when events happen. 

#### Maxmind
- Implement auto-updating databases

#### Usage
- Implement better usage tracking (Tracking Domain requested, Member assigned, ASN, City, State, Country and postal of Request, Class C of request, ISP, Connection Type)
 
#### Monitor
- Implement genesis hash check

#### Billing

#### PUB SUB
- Members
  - Handle member status request from GW
  - Handle member activate request from GW
  - Handle member deactivate request from GW
  - Handle member events request from GW
- Stats
  - Handle stats request from GW
- Usage
  - Handle usage request from GW
- Synchronize events data between nodes over pubsub

## Member & User Gateway

- Figure out a way to allow members to receive push notifications when events happen. 

#### API
- Build standalone rest GW
- Members
  - Get members list
  - Get member details
  - Get member events
  - Set member maintenance (enable/disable)
  - Change member api key
- Services
  - Get service levels
  - Get rpc services
  - Get bootnode services
- Billing
  - Get finalized billing for period
  - Get IaaS component pricing
  - Get estimated annual and monthly pricing
- Stats
  - Get all stats for period (Flags: Domain, Member, Country, ASN, IP Block)
- Cache
  - Implement stats and usage collection on GW to reduce work on nodes

#### Matrix
- Build standalone matrix GW
- Members
  - Get members list
  - Get member details
  - Get member events
  - Set member maintenance (enable/disable)
  - Change member api key
- Services
  - Get service levels
  - Get rpc services
  - Get bootnode services
- Billing
  - Get finalized billing for period
  - Get IaaS component pricing
  - Get estimated annual and monthly pricing
- Stats
  - Get all stats for period (Flags: Domain, Member, Country, ASN, IP Block)
- Cache
  - Implement stats and usage collection on GW to reduce work on nodes