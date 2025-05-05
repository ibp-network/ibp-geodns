package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	max "ibp-geodns/src/common/maxmind"

	"golang.org/x/net/publicsuffix"
)

// DynamicDNSEntries remains similar, it populates the map of domains and members.
func DynamicDNSEntries() {
	c := cfg.GetConfig()

	newDynamicServices := make(map[string]ServiceConfigs)

	// Populate service-domain map from services
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

	// Assign members to services
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

	// Read service configs
	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for serviceDomain, service := range ServiceRecords.Services {
		if serviceDomain == domain {
			for _, member := range service.Members {

				// If member override is on skip member
				if member.Override {
					continue
				}

				// Check if member is online using official data:
				if !IsValidIPv4(member.Service.ServiceIPv4) {
					continue
				}

				// Use data package to determine online/offline:
				if !dat.IsMemberOnlineForDomain(domain, member.Details.Name) {
					continue
				}

				// If the member is online, measure distance and pick closest
				dist := max.Distance(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
				if dist < minDistance {
					minDistance = dist
					closestMember = member
				}
			}

			// If we found a closest member
			if closestMember.Details.Name != "" {
				// Append IPv4 records if requested
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

				// Append IPv6 if available (adjust if needed)
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

				// Store Member Stats
				if closestMember.Details.Name != "" {
					go dat.MemberHit(closestMember.Details.Name, params.Remote, domain)
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

// GenerateTLDs iterates through StaticDNS and Services to collect unique top-level domains
func GenerateTLDs() {
	c := cfg.GetConfig()

	// Initialize the map with integer keys and string values
	TLDRecords.records = make(map[int]string)

	// Initialize a unique integer key
	key := 0

	// Use a map to keep track of domains to avoid duplicates
	domainSet := make(map[string]struct{})

	// Iterate over StaticDNS entries
	for _, dnsRecord := range c.StaticDNS {
		tld := extractTopLevelDomain(dnsRecord.QName)
		if tld != "" {
			if _, exists := domainSet[tld]; !exists {
				key++
				TLDRecords.records[key] = tld
				domainSet[tld] = struct{}{}
			}
		}
	}

	// Iterate over Services' RPC URLs
	for _, service := range c.Services {
		for _, provider := range service.Providers {
			for _, rpcURL := range provider.RpcUrls {
				tld := extractTopLevelDomain(rpcURL)
				if tld != "" {
					if _, exists := domainSet[tld]; !exists {
						key++
						TLDRecords.records[key] = tld
						domainSet[tld] = struct{}{}
					}
				}
			}
		}
	}
}

// Helper function to extract the registrable domain (eTLD+1)
func extractTopLevelDomain(domain string) string {
	// Remove protocol if present
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "wss://")
	domain = strings.TrimPrefix(domain, "ws://")

	// Remove any trailing path or query parameters
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}

	// Convert to lowercase and trim spaces
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Extract the registrable domain (eTLD+1)
	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		// Handle error or return empty string if domain is invalid
		return ""
	}
	return eTLDPlusOne
}
