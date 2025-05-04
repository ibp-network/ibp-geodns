package consensus

import (
	"strings"

	"ibp-geodns/src/common/config"
	g "ibp-geodns/src/common/maxmind"
)

// findMemberByName searches for a member by name.
func findMemberByName(memberName string) (config.Member, bool) {
	c := config.GetConfig()
	for _, m := range c.Members {
		if m.Details.Name == memberName {
			return m, true
		}
	}
	return config.Member{}, false
}

// findCheckByName searches for a check by name and type.
func findCheckByName(checkName, checkType string) (config.Check, bool) {
	c := config.GetConfig()
	for _, ch := range c.System.Checks {
		if ch.Name == checkName && ch.CheckType == checkType {
			return ch, true
		}
	}
	return config.Check{}, false
}

// findServiceForDomain finds a service that matches the given domainName.
func findServiceForDomain(domainName string) (config.Service, bool) {
	c := config.GetConfig()
	for _, service := range c.Services {
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				u := g.ParseUrl(rpcUrl)
				if strings.EqualFold(u.Domain, domainName) {
					return service, true
				}
			}
		}
	}
	return config.Service{}, false
}
