package api

import (
	"strings"

	max "github.com/ibp-network/ibp-geodns-libs/maxmind"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func RebuildServiceRecords() {
	log.Log(log.Info, "RebuildServiceRecords: rebuilding dynamic ServiceRecords from config...")

	ServiceRecords.mu.Lock()

	c := cfg.GetConfig()

	newMap := make(map[string]ServiceConfigs)

	for _, svc := range c.Services {
		if svc.Configuration.Active == 0 {
			continue
		}
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				parsed := max.ParseUrl(rpcUrl)
				domain := strings.ToLower(parsed.Domain)

				sc, found := newMap[domain]
				if !found {
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

	for memberName, member := range c.Members {
		for svcName, svc := range c.Services {
			if isServiceAssignedToMember(svcName, member) {
				for _, provider := range svc.Providers {
					for _, rpcUrl := range provider.RpcUrls {
						parsed := max.ParseUrl(rpcUrl)
						domain := strings.ToLower(parsed.Domain)

						sc, ok := newMap[domain]
						if !ok {
							continue
						}
						sc.Members[memberName] = member
						newMap[domain] = sc
					}
				}
			}
		}
	}

	ServiceRecords.Services = newMap

	log.Log(log.Info, "RebuildServiceRecords: now have %d domain(s) in ServiceRecords", len(ServiceRecords.Services))
	for dom, sc := range ServiceRecords.Services {
		log.Log(log.Info, " - domain=%s => %d assigned members", dom, len(sc.Members))
	}

	ServiceRecords.mu.Unlock()

	// refresh TLD records to reflect dynamic domain changes (must be after unlocking)
	populateTLDRecords()
}

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
