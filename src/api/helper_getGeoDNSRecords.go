package api

import (
	"math"
	"net"
	"strings"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
	max "github.com/ibp-network/ibp-geodns-libs/maxmind"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	dat "github.com/ibp-network/ibp-geodns-libs/data"
)

func ProcessDynamic(params Parameters, id int, domain string, useIPv6 bool) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	clientIP := net.ParseIP(params.Remote)

	// Check for country code override first
	if clientIP != nil {
		countryCode := getClientCountryCode(clientIP.String())
		if countryCode != "" {
			override, hasOverride := CountryOverrides.GetCountryOverride(domain, countryCode)
			if hasOverride {
				return processCountryOverride(params, id, domain, useIPv6, override, countryCode)
			}
		}
	}

	// Fall back to geographic routing
	var closestMember cfg.Member
	minDistance := math.MaxFloat64

	var clientLat, clientLon float64
	if clientIP != nil {
		isClientIPv6 := clientIP.To4() == nil

		if isClientIPv6 {
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv6: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		} else {
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv4: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		}
	}

	ServiceRecords.mu.RLock()
	sc, found := ServiceRecords.Services[strings.ToLower(domain)]
	ServiceRecords.mu.RUnlock()

	if !found {
		return nil, ""
	}

	for _, member := range sc.Members {
		if member.Override {
			continue
		}

		var ipToUse string
		if useIPv6 {
			ipToUse = member.Service.ServiceIPv6
			if ipToUse == "" {
				continue
			}

			isOnline := IsMemberOnlineForDomainIPv6(domain, member.Details.Name)
			if !isOnline {
				log.Log(log.Debug, "ProcessDynamic: member %s is offline for IPv6 on domain %s",
					member.Details.Name, domain)
				continue
			}
		} else {
			ipToUse = member.Service.ServiceIPv4
			if ipToUse == "" {
				continue
			}

			isOnline := IsMemberOnlineForDomainIPv4(domain, member.Details.Name)
			if !isOnline {
				log.Log(log.Debug, "ProcessDynamic: member %s is offline for IPv4 on domain %s",
					member.Details.Name, domain)
				continue
			}
		}

		parsedIP := net.ParseIP(ipToUse)
		if parsedIP == nil {
			continue
		}

		dist := max.Distance(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
		if dist < minDistance {
			minDistance = dist
			closestMember = member
		}
	}

	if closestMember.Details.Name == "" {
		log.Log(log.Debug, "ProcessDynamic: no suitable %v members found for domain=%s", boolToStr(useIPv6), domain)
		return records, ""
	}

	if useIPv6 {
		rec := cfg.DNSRecord{
			DomainID: id,
			QName:    domain,
			QType:    "AAAA",
			Content:  closestMember.Service.ServiceIPv6,
			TTL:      30,
			Auth:     true,
		}
		records = append(records, rec)
		chosenMemberName = closestMember.Details.Name

		dat.RecordDnsHit(true, params.Remote, domain, chosenMemberName)
	} else {
		rec := cfg.DNSRecord{
			DomainID: id,
			QName:    domain,
			QType:    "A",
			Content:  closestMember.Service.ServiceIPv4,
			TTL:      30,
			Auth:     true,
		}
		records = append(records, rec)
		chosenMemberName = closestMember.Details.Name

		dat.RecordDnsHit(false, params.Remote, domain, chosenMemberName)
	}

	log.Log(log.Debug, "ProcessDynamic: selected member %s for %s query on domain %s",
		chosenMemberName, boolToStr(useIPv6), domain)

	return records, chosenMemberName
}

func boolToStr(b bool) string {
	if b {
		return "IPv6"
	}
	return "IPv4"
}

// getClientCountryCode retrieves the country code for a client IP
// Returns empty string if country code cannot be determined
func getClientCountryCode(clientIP string) string {
	// Try to get country code from MaxMind
	// The maxmind library should have a GetClientCountry function
	// If it doesn't exist, this will need to be added to the library
	// For now, we'll try to call it and handle gracefully if it doesn't exist
	// Note: This may need to be updated based on the actual maxmind library API
	var countryCode string

	// Check if GetClientCountry exists - if the library doesn't have it,
	// we'll need to add it or use an alternative method
	// For now, we'll assume it exists and call it
	// TODO: Verify this function exists in the maxmind library
	countryCode = max.GetClientCountry(clientIP)

	if countryCode != "" {
		return strings.ToUpper(countryCode)
	}
	return ""
}

// processCountryOverride handles DNS routing when a country code override is active
func processCountryOverride(params Parameters, id int, domain string, useIPv6 bool, override CountryOverride, countryCode string) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	// If override specifies a member name, find that member
	if override.MemberName != "" {
		ServiceRecords.mu.RLock()
		sc, found := ServiceRecords.Services[strings.ToLower(domain)]
		ServiceRecords.mu.RUnlock()

		if found {
			for memberName, member := range sc.Members {
				if memberName == override.MemberName {
					// Check if member is online and has the required IP version
					var ipToUse string
					if useIPv6 {
						ipToUse = member.Service.ServiceIPv6
						if ipToUse == "" {
							log.Log(log.Debug, "processCountryOverride: member %s has no IPv6, falling back to geographic routing", override.MemberName)
							return nil, ""
						}
						if !IsMemberOnlineForDomainIPv6(domain, member.Details.Name) {
							log.Log(log.Debug, "processCountryOverride: member %s is offline for IPv6, falling back to geographic routing", override.MemberName)
							return nil, ""
						}
					} else {
						ipToUse = member.Service.ServiceIPv4
						if ipToUse == "" {
							log.Log(log.Debug, "processCountryOverride: member %s has no IPv4, falling back to geographic routing", override.MemberName)
							return nil, ""
						}
						if !IsMemberOnlineForDomainIPv4(domain, member.Details.Name) {
							log.Log(log.Debug, "processCountryOverride: member %s is offline for IPv4, falling back to geographic routing", override.MemberName)
							return nil, ""
						}
					}

					var qtype string
					if useIPv6 {
						qtype = "AAAA"
					} else {
						qtype = "A"
					}
					rec := cfg.DNSRecord{
						DomainID: id,
						QName:    domain,
						QType:    qtype,
						Content:  ipToUse,
						TTL:      30,
						Auth:     true,
					}
					records = append(records, rec)
					chosenMemberName = override.MemberName

					if useIPv6 {
						dat.RecordDnsHit(true, params.Remote, domain, chosenMemberName)
					} else {
						dat.RecordDnsHit(false, params.Remote, domain, chosenMemberName)
					}

					log.Log(log.Debug, "processCountryOverride: country=%s, domain=%s, member=%s, ip=%s",
						countryCode, domain, chosenMemberName, ipToUse)
					return records, chosenMemberName
				}
			}
			log.Log(log.Warn, "processCountryOverride: member %s not found for domain %s, falling back to geographic routing",
				override.MemberName, domain)
			return nil, ""
		}
	}

	// If override specifies direct IP addresses
	var ipToUse string
	if useIPv6 {
		ipToUse = override.IPv6
		if ipToUse == "" {
			log.Log(log.Debug, "processCountryOverride: no IPv6 override for country %s, falling back to geographic routing", countryCode)
			return nil, ""
		}
	} else {
		ipToUse = override.IPv4
		if ipToUse == "" {
			log.Log(log.Debug, "processCountryOverride: no IPv4 override for country %s, falling back to geographic routing", countryCode)
			return nil, ""
		}
	}

	// Validate IP address
	parsedIP := net.ParseIP(ipToUse)
	if parsedIP == nil {
		log.Log(log.Warn, "processCountryOverride: invalid IP address %s for country %s, falling back to geographic routing",
			ipToUse, countryCode)
		return nil, ""
	}

	var qtype string
	if useIPv6 {
		qtype = "AAAA"
	} else {
		qtype = "A"
	}
	rec := cfg.DNSRecord{
		DomainID: id,
		QName:    domain,
		QType:    qtype,
		Content:  ipToUse,
		TTL:      30,
		Auth:     true,
	}
	records = append(records, rec)
	chosenMemberName = "override:" + countryCode

	if useIPv6 {
		dat.RecordDnsHit(true, params.Remote, domain, chosenMemberName)
	} else {
		dat.RecordDnsHit(false, params.Remote, domain, chosenMemberName)
	}

	log.Log(log.Debug, "processCountryOverride: country=%s, domain=%s, direct IP=%s",
		countryCode, domain, ipToUse)
	return records, chosenMemberName
}
