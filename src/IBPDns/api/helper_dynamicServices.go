package api

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"strings"
)

func RebuildServiceRecords() {
	log.Log(log.Info, "RebuildServiceRecords: rebuilding dynamic ServiceRecords from config...")

	ServiceRecords.mu.Lock()
	defer ServiceRecords.mu.Unlock()

	c := cfg.GetConfig()

	newMap := make(map[string]ServiceConfigs)

	for _, svc := range c.Services {
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
