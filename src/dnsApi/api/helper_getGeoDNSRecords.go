package api

import (
	"math"
	"net"

	cfg "ibp-geodns/src/common/config"
	max "ibp-geodns/src/common/maxmind"
)

// DynamicDNSEntries remains the same
func DynamicDNSEntries() {
	c := cfg.GetConfig()

	newDynamicServices := make(map[string]ServiceConfigs)

	for _, service := range c.Services {
		svcConfig := service.Configuration
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				url := max.ParseUrl(rpcUrl)
				if _, exists := newDynamicServices[url.Domain]; !exists {
					newDynamicServices[url.Domain] = ServiceConfigs{
						Name:          svcConfig.Name,
						Active:        svcConfig.Active,
						LevelRequired: svcConfig.LevelRequired,
						NetworkName:   svcConfig.NetworkName,
						Members:       make(map[string]cfg.Member),
					}
				}
			}
		}
	}

	for _, member := range c.Members {
		if member.Service.Active != 1 {
			continue
		}
		for _, assignments := range member.ServiceAssignments {
			for _, assignment := range assignments {
				for _, service := range c.Services {
					if assignment == service.Configuration.Name {
						if member.Membership.Level < service.Configuration.LevelRequired {
							continue
						}
						for domainName, serviceConfig := range newDynamicServices {
							if serviceConfig.Name == service.Configuration.Name {
								memberInfo := cfg.Member{
									Details:    member.Details,
									Membership: member.Membership,
									Service:    member.Service,
									Location:   member.Location,
								}
								newDynamicServices[domainName].Members[member.Details.Name] = memberInfo
							}
						}
					}
				}
			}
		}
	}

	ServiceRecords.mu.Lock()
	defer ServiceRecords.mu.Unlock()
	ServiceRecords.Services = newDynamicServices
}

func ProcessDynamic(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	var closestMember cfg.Member
	minDistance := math.MaxFloat64
	clientLat, clientLon := max.GetClientCoordinates(params.Remote)

	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for serviceDomain, service := range ServiceRecords.Services {
		if serviceDomain == domain {
			for _, member := range service.Members {
				// override skip
				if member.Override {
					continue
				}
				// check if online
				if !IsValidIPv4(member.Service.ServiceIPv4) {
					continue
				}
				// use the new officialResults approach
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
	return records
}

func IsValidIPv4(ip string) bool {
	parsedIP := net.ParseIP(ip)
	return parsedIP != nil && parsedIP.To4() != nil
}
