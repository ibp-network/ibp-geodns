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

				// The "Name" might not match the domain, but we can store it if you want
				// sc.Name = svc.Configuration.Name
				// sc.NetworkName = svc.Configuration.NetworkName

				newMap[domain] = sc
			}
		}
	}

	// Now we must figure out which members are assigned to which domain(s).
	// Each member can have "ServiceAssignments" = map[string][]string
	// e.g. member.ServiceAssignments["dotters"] = ["hydra", "mythos"]
	// Then c.Services["mythos"] might lead us to "mythos.dotters.network" domain, etc.

	// We'll cross-match members' assignments with the newMap so that newMap[domain].Members
	// includes that member.

	for memberName, member := range c.Members {
		// The member has a map of <whatever> => []strings
		// Example: member.ServiceAssignments["dotters"] = ["mythos","hydration","kusa"]
		// We need to see if that matches any of c.Services keys.

		// We'll scan the entire newMap for each serviceName:
		// but a simpler approach might be scanning c.Services again.

		// For each (svcName, svc) in c.Services:
		//   if svcName is in the member's assignment list => that member should be part of the domain(s).

		for svcName, svc := range c.Services {
			// Check if this member is assigned that service
			if isServiceAssignedToMember(svcName, member) {
				// Then all domain(s) from that service must have this member
				for _, provider := range svc.Providers {
					for _, rpcUrl := range provider.RpcUrls {
						parsed := max.ParseUrl(rpcUrl)
						domain := strings.ToLower(parsed.Domain)

						sc, ok := newMap[domain]
						if !ok {
							// Possibly the domain wasn't recognized for some reason,
							// but theoretically it should exist from above
							continue
						}

						// Insert/Update the member in sc.Members
						sc.Members[memberName] = member
						// Reassign updated sc
						newMap[domain] = sc
					}
				}
			}
		}
	}

	// Finally, lock & replace ServiceRecords.Services
	ServiceRecords.mu.Lock()
	defer ServiceRecords.mu.Unlock()
	ServiceRecords.Services = newMap

	log.Log(log.Info, "RebuildServiceRecords: now have %d domain(s) in ServiceRecords", len(ServiceRecords.Services))
	for dom, sc := range ServiceRecords.Services {
		log.Log(log.Info, " - domain=%s => %d assigned members", dom, len(sc.Members))
	}
}

// isServiceAssignedToMember checks if the given serviceName is in the member's ServiceAssignments.
func isServiceAssignedToMember(serviceName string, member cfg.Member) bool {
	// member.ServiceAssignments might be something like:
	//  {
	//    "dotters": ["mythos","hydration"],
	//    "kusama": ["rpc1","rpc2"]
	//  }
	// We want to see if 'serviceName' is in ANY of the slices in that map,
	// or maybe a direct string match if your data is structured differently.

	for _, assignmentList := range member.ServiceAssignments {
		for _, val := range assignmentList {
			if val == serviceName {
				return true
			}
		}
	}
	return false
}
