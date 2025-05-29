package api

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"strings"
)

// RebuildServiceRecords scans the config and populates ServiceRecords.Services
// with domain -> {Name, Active, LevelRequired, etc.} plus the relevant Members.
func RebuildServiceRecords() {
	log.Log(log.Info, "RebuildServiceRecords: rebuilding dynamic ServiceRecords from config...")

	// Lock the ServiceRecords for the entire process to ensure concurrency safety.
	ServiceRecords.mu.Lock()
	defer ServiceRecords.mu.Unlock()

	c := cfg.GetConfig()

	// Prepare a fresh map
	newMap := make(map[string]ServiceConfigs)

	// We'll iterate over c.Services and find all possible domains from each service
	// by examining each service's Providers => each provider's RpcUrls => domain from that URL.
	for _, svc := range c.Services {
		// Each service can have multiple providers, each with multiple URLs
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				parsed := max.ParseUrl(rpcUrl)
				domain := strings.ToLower(parsed.Domain)

				// See if we already have a ServiceConfigs for that domain
				sc, found := newMap[domain]
				if !found {
					// Initialize it
					sc = ServiceConfigs{
						Name:          svc.Configuration.Name,
						Active:        svc.Configuration.Active,
						LevelRequired: svc.Configuration.LevelRequired,
						NetworkName:   svc.Configuration.NetworkName,
						ServiceType:   svc.Configuration.ServiceType,
						Members:       make(map[string]cfg.Member),
					}
				}

				newMap[domain] = sc
			}
		}
	}

	// Now figure out which members are assigned to which domain(s).
	// Each member has ServiceAssignments, e.g. { "dotters": ["mythos"] }.
	for memberName, member := range c.Members {
		for svcName, svc := range c.Services {
			if isServiceAssignedToMember(svcName, member) {
				// Then all domain(s) from that service must have this member
				for _, provider := range svc.Providers {
					for _, rpcUrl := range provider.RpcUrls {
						parsed := max.ParseUrl(rpcUrl)
						domain := strings.ToLower(parsed.Domain)

						sc, ok := newMap[domain]
						if !ok {
							// Possibly the domain wasn't recognized, but in theory it should exist.
							continue
						}
						sc.Members[memberName] = member
						newMap[domain] = sc
					}
				}
			}
		}
	}

	// Assign the newly built map
	ServiceRecords.Services = newMap

	log.Log(log.Info, "RebuildServiceRecords: now have %d domain(s) in ServiceRecords", len(ServiceRecords.Services))
	for dom, sc := range ServiceRecords.Services {
		log.Log(log.Info, " - domain=%s => %d assigned members", dom, len(sc.Members))
	}
}

// isServiceAssignedToMember checks if the given serviceName is in the member's ServiceAssignments.
func isServiceAssignedToMember(serviceName string, member cfg.Member) bool {
	for _, assignmentList := range member.ServiceAssignments {
		for _, val := range assignmentList {
			if val == serviceName {
				return true
			}
		}
	}
	return false
}
