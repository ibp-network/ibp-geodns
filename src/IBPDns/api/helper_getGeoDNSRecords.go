package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

// ProcessDynamic decides whether we want IPv4 or IPv6 (based on `useIPv6`) and picks the best online member.
func ProcessDynamic(params Parameters, id int, domain string, useIPv6 bool) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	var closestMember cfg.Member
	minDistance := math.MaxFloat64

	clientIP := net.ParseIP(params.Remote)
	var clientLat, clientLon float64
	if clientIP != nil {
		// Use the real MaxMind call to get client coordinates
		clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
	}

	// Acquire the service configuration
	ServiceRecords.mu.RLock()
	sc, found := ServiceRecords.Services[strings.ToLower(domain)]
	ServiceRecords.mu.RUnlock()

	if !found {
		// No dynamic config for this domain
		return nil, ""
	}

	for _, member := range sc.Members {
		// Skip if override is set
		if member.Override {
			continue
		}

		// Decide which IP (v4 or v6) to use
		var ipToUse string
		if useIPv6 {
			ipToUse = member.Service.ServiceIPv6
			if ipToUse == "" {
				continue
			}
			// Official check for IPv6
			isOnline := IsMemberOnlineForDomainIPv4v6(domain, member.Details.Name, true)
			if !isOnline {
				continue
			}
		} else {
			ipToUse = member.Service.ServiceIPv4
			if ipToUse == "" {
				continue
			}
			// Official check for IPv4
			isOnline := IsMemberOnlineForDomainIPv4v6(domain, member.Details.Name, false)
			if !isOnline {
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
		// None matched
		log.Log(log.Debug, "ProcessDynamic: no suitable %v members found for domain=%s", boolToStr(useIPv6), domain)
		return records, ""
	}

	// Produce final record
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
	}

	return records, chosenMemberName
}

// IsMemberOnlineForDomainIPv4v6 checks official data for the given domain and IP family
// FIXED to actually call the correct IPv4 vs. IPv6 checks.
func IsMemberOnlineForDomainIPv4v6(domain, memberName string, useIPv6 bool) bool {
	if useIPv6 {
		return dat.IsMemberOnlineForDomainIPv6(domain, memberName)
	}
	return dat.IsMemberOnlineForDomain(domain, memberName)
}

func boolToStr(b bool) string {
	if b {
		return "IPv6"
	}
	return "IPv4"
}
