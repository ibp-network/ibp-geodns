package api

import (
	"strings"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

// SetCountryOverride sets or updates a country code override for a domain
func (com *CountryOverrideMap) SetCountryOverride(domain, countryCode string, override CountryOverride) {
	com.mu.Lock()
	defer com.mu.Unlock()

	domain = strings.ToLower(domain)
	countryCode = strings.ToUpper(countryCode)

	if com.overrides[domain] == nil {
		com.overrides[domain] = make(map[string]CountryOverride)
	}

	com.overrides[domain][countryCode] = override
	log.Log(log.Info, "SetCountryOverride: domain=%s, country=%s, member=%s, ipv4=%s, ipv6=%s",
		domain, countryCode, override.MemberName, override.IPv4, override.IPv6)
}

// RemoveCountryOverride removes a country code override for a domain
func (com *CountryOverrideMap) RemoveCountryOverride(domain, countryCode string) {
	com.mu.Lock()
	defer com.mu.Unlock()

	domain = strings.ToLower(domain)
	countryCode = strings.ToUpper(countryCode)

	if com.overrides[domain] != nil {
		delete(com.overrides[domain], countryCode)
		if len(com.overrides[domain]) == 0 {
			delete(com.overrides, domain)
		}
		log.Log(log.Info, "RemoveCountryOverride: domain=%s, country=%s", domain, countryCode)
	}
}

// GetCountryOverride retrieves a country code override for a domain
func (com *CountryOverrideMap) GetCountryOverride(domain, countryCode string) (CountryOverride, bool) {
	com.mu.RLock()
	defer com.mu.RUnlock()

	domain = strings.ToLower(domain)
	countryCode = strings.ToUpper(countryCode)

	if domainOverrides, ok := com.overrides[domain]; ok {
		if override, ok := domainOverrides[countryCode]; ok {
			return override, true
		}
	}
	return CountryOverride{}, false
}

// LoadCountryOverridesFromConfig loads country overrides from the config structure
// This expects a map structure like: map[domain]map[countryCode]CountryOverride
func LoadCountryOverridesFromConfig(configOverrides map[string]map[string]CountryOverride) {
	CountryOverrides.mu.Lock()
	defer CountryOverrides.mu.Unlock()

	// Clear existing overrides
	CountryOverrides.overrides = make(map[string]map[string]CountryOverride)

	// Load new overrides
	for domain, domainOverrides := range configOverrides {
		domain = strings.ToLower(domain)
		CountryOverrides.overrides[domain] = make(map[string]CountryOverride)
		for countryCode, override := range domainOverrides {
			countryCode = strings.ToUpper(countryCode)
			CountryOverrides.overrides[domain][countryCode] = override
		}
	}

	log.Log(log.Info, "LoadCountryOverridesFromConfig: loaded overrides for %d domain(s)", len(CountryOverrides.overrides))
	for domain, domainOverrides := range CountryOverrides.overrides {
		log.Log(log.Info, "  - domain=%s: %d country override(s)", domain, len(domainOverrides))
	}
}
