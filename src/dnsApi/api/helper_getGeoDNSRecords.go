package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	max "ibp-geodns/src/common/maxmind"
)

// ProcessDynamic chooses the closest online member for the given domain
// and returns (records, chosenMemberName). It uses the IsMemberOnlineForDomain()
// function, which references the official results snapshot.
func ProcessDynamic(params Parameters, id int, domain string) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	var closestMember cfg.Member
	minDistance := math.MaxFloat64
	clientLat, clientLon := max.GetClientCoordinates(params.Remote)

	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for serviceDomain, serviceConfig := range ServiceRecords.Services {
		// For domain matches exactly
		if strings.EqualFold(serviceDomain, domain) {
			for _, member := range serviceConfig.Members {
				if member.Override {
					continue
				}
				if !IsValidIPv4(member.Service.ServiceIPv4) {
					continue
				}

				// check official results
				if !IsMemberOnlineForDomain(domain, member.Details.Name) {
					continue
				}

				dist := max.Distance(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
				if dist < minDistance {
					minDistance = dist
					closestMember = member
				}
			}
			if closestMember.Details.Name != "" {
				chosenMemberName = closestMember.Details.Name

				if params.QType == "A" || params.QType == "ANY" {
					if closestMember.Service.ServiceIPv4 != "" {
						records = append(records, cfg.DNSRecord{
							DomainID: id,
							QName:    domain,
							QType:    "A",
							Content:  closestMember.Service.ServiceIPv4,
							TTL:      30,
							Auth:     true,
						})
					}
				}
				if params.QType == "AAAA" || params.QType == "ANY" {
					if closestMember.Service.ServiceIPv6 != "" {
						records = append(records, cfg.DNSRecord{
							DomainID: id,
							QName:    domain,
							QType:    "AAAA",
							Content:  closestMember.Service.ServiceIPv6,
							TTL:      30,
							Auth:     true,
						})
					}
				}
			}
		}
	}

	return records, chosenMemberName
}

// IsValidIPv4 checks if a string is a valid IPv4 address
func IsValidIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() != nil
}
