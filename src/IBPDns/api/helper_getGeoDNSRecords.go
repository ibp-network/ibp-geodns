package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	max "ibp-geodns/src/common/maxmind"
)

// ProcessDynamic chooses the closest online member for the given domain
// and returns (records, chosenMemberName). The `useIPv6` param indicates whether
// we want to pick IPv6 addresses (AAAA) or IPv4 addresses (A).
func ProcessDynamic(params Parameters, id int, domain string, useIPv6 bool) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	var closestMember cfg.Member
	minDistance := math.MaxFloat64

	clientLat, clientLon := max.GetClientCoordinates(params.Remote)

	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for serviceDomain, serviceConfig := range ServiceRecords.Services {
		if strings.EqualFold(serviceDomain, domain) {
			// Evaluate each assigned member
			for _, member := range serviceConfig.Members {
				// Must not be forcibly overridden offline
				if member.Override {
					continue
				}

				// Check if we want IPv4 or IPv6
				addrToUse := ""
				if useIPv6 {
					if member.Service.ServiceIPv6 == "" {
						continue
					}
					// Official checks for IPv6
					if !IsMemberOnlineForDomainV6(domain, member.Details.Name) {
						continue
					}
					addrToUse = member.Service.ServiceIPv6
				} else {
					if member.Service.ServiceIPv4 == "" {
						continue
					}
					// Official checks for IPv4
					if !IsMemberOnlineForDomain(domain, member.Details.Name) {
						continue
					}
					addrToUse = member.Service.ServiceIPv4
				}

				// Validate IP
				testIP := net.ParseIP(addrToUse)
				if testIP == nil {
					continue
				}

				dist := max.Distance(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
				if dist < minDistance {
					minDistance = dist
					closestMember = member
				}
			}

			// If we have found a best match
			if closestMember.Details.Name != "" {
				chosenMemberName = closestMember.Details.Name
				if useIPv6 {
					// AAAA
					rec := cfg.DNSRecord{
						DomainID: id,
						QName:    domain,
						QType:    "AAAA",
						Content:  closestMember.Service.ServiceIPv6,
						TTL:      30,
						Auth:     true,
					}
					records = append(records, rec)
				} else {
					// A
					rec := cfg.DNSRecord{
						DomainID: id,
						QName:    domain,
						QType:    "A",
						Content:  closestMember.Service.ServiceIPv4,
						TTL:      30,
						Auth:     true,
					}
					records = append(records, rec)
				}
			}
		}
	}

	return records, chosenMemberName
}

// IsMemberOnlineForDomainV6 checks official results for IPv6 by calling
// our data-layer function that filters on IPv6 only.
func IsMemberOnlineForDomainV6(domain, memberName string) bool {
	return dat.IsMemberOnlineForDomainIPv6(domain, memberName)
}

// IsValidIPv4 checks if a string is a valid IPv4 address
func IsValidIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() != nil
}
