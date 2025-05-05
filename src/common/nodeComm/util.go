package nodeComm

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	max "ibp-geodns/src/common/maxmind"
)

// findMemberByName searches for a member by name.
func findMemberByName(memberName string) (cfg.Member, bool) {
	c := cfg.GetConfig()
	for _, m := range c.Members {
		if m.Details.Name == memberName {
			return m, true
		}
	}
	return cfg.Member{}, false
}

// findCheckByName searches for a check by name and type.
func findCheckByName(checkName, checkType string) (cfg.Check, bool) {
	c := cfg.GetConfig()
	for _, ch := range c.System.Checks {
		if ch.Name == checkName && ch.CheckType == checkType {
			return ch, true
		}
	}
	return cfg.Check{}, false
}

// findServiceForDomain finds a service that matches the given domainName.
func findServiceForDomain(domainName string) (cfg.Service, bool) {
	c := cfg.GetConfig()
	for _, service := range c.Services {
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				u := max.ParseUrl(rpcUrl)
				if strings.EqualFold(u.Domain, domainName) {
					return service, true
				}
			}
		}
	}
	return cfg.Service{}, false
}
