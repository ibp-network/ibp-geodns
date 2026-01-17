# GeoDNS Country Code Overrides

This document describes the country code override feature that allows routing specific country codes to designated endpoints.

## Overview

The GeoDNS override system allows you to:
- Route specific country codes (e.g., CN, JP) to specific member endpoints or IP addresses
- Override the default geographic routing for certain countries
- Update overrides at runtime via NATS messages
- Configure overrides statically in the config file

## Configuration

### Static Configuration

Add a `GeoDNSOverrides` section to your `ibpdns.json` config file:

```json
{
  "GeoDNSOverrides": {
    "example.com": {
      "CN": {
        "memberName": "member-china",
        "ipv4": "",
        "ipv6": ""
      },
      "JP": {
        "memberName": "",
        "ipv4": "203.0.113.10",
        "ipv6": "2001:db8::10"
      }
    }
  }
}
```

**Structure:**
- Top level: Domain name (e.g., "example.com")
- Second level: Country code (ISO 3166-1 alpha-2, e.g., "CN", "JP")
- Override object:
  - `memberName` (optional): Name of the member to route to (takes precedence over IPs)
  - `ipv4` (optional): Direct IPv4 address to route to
  - `ipv6` (optional): Direct IPv6 address to route to

**Priority:**
1. If `memberName` is specified, the system will route to that member (if online and healthy)
2. If `memberName` is empty but IP addresses are specified, those IPs will be used directly
3. If no override matches, the system falls back to normal geographic routing

## Runtime Updates via NATS

You can update overrides at runtime by publishing messages to the NATS subject: `geodns.override.update`

### Set Override

```json
{
  "action": "set",
  "domain": "example.com",
  "countryCode": "CN",
  "override": {
    "memberName": "member-china",
    "ipv4": "",
    "ipv6": ""
  }
}
```

### Remove Override

```json
{
  "action": "remove",
  "domain": "example.com",
  "countryCode": "CN"
}
```

### Clear All Overrides for Domain

```json
{
  "action": "clear",
  "domain": "example.com"
}
```

## How It Works

1. When a DNS query arrives, the system first checks the client's country code using MaxMind GeoIP
2. If a country code override exists for that domain and country, it uses the override
3. If the override specifies a `memberName`, it routes to that member (if online)
4. If the override specifies direct IPs, it uses those IPs
5. If no override matches or the override target is unavailable, it falls back to normal geographic routing

## Implementation Details

- Overrides are stored in thread-safe maps
- Country codes are automatically normalized to uppercase
- Domain names are automatically normalized to lowercase
- Overrides are checked before geographic distance calculations
- Health checks are still performed for member-based overrides

## Requirements

**Note:** This implementation assumes the MaxMind library has a `GetClientCountry(ip string) string` function. If this function doesn't exist in your MaxMind library, you'll need to:

1. Add it to the `ibp-geodns-libs/maxmind` package, or
2. Modify `getClientCountryCode()` in `src/api/helper_getGeoDNSRecords.go` to use an alternative method

The function should return the ISO 3166-1 alpha-2 country code (e.g., "CN", "US", "JP") for a given IP address.

## Example Use Cases

1. **Route China traffic to a specific member:**
   ```json
   "CN": {
     "memberName": "member-china"
   }
   ```

2. **Route Japan traffic to a specific IP:**
   ```json
   "JP": {
     "ipv4": "203.0.113.10",
     "ipv6": "2001:db8::10"
   }
   ```

3. **Route multiple countries to different endpoints:**
   ```json
   "example.com": {
     "CN": { "memberName": "member-china" },
     "JP": { "memberName": "member-japan" },
     "KR": { "ipv4": "203.0.113.20" }
   }
   ```

## Testing

To test the override system:

1. Add an override to your config file
2. Restart the DNS service
3. Query from an IP in the overridden country
4. Verify the response uses the override endpoint
5. Test runtime updates via NATS messages
