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
		// Check if the client IP is IPv6
		isClientIPv6 := clientIP.To4() == nil

		// Use appropriate geolocation based on client IP version
		if isClientIPv6 {
			// For IPv6 clients, we should ideally have a separate IPv6 geolocation method
			// For now, we'll use the same method but log it
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv6: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		} else {
			// IPv4 client
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv4: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		}
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
			// FIXED: Use IPv6-specific status check
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
			// FIXED: Use IPv4-specific status check
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

		// Record the DNS hit for usage stats
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

		// Record the DNS hit for usage stats
		dat.RecordDnsHit(false, params.Remote, domain, chosenMemberName)
	}

	log.Log(log.Debug, "ProcessDynamic: selected member %s for %s query on domain %s",
		chosenMemberName, boolToStr(useIPv6), domain)

	return records, chosenMemberName
}

// REMOVED: IsMemberOnlineForDomainIPv4v6 - this was the problematic function
// Now we use the specific IPv4/IPv6 functions from helper_monitor.go

func boolToStr(b bool) string {
	if b {
		return "IPv6"
	}
	return "IPv4"
}
