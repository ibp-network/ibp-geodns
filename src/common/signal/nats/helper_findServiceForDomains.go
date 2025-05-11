package nats

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	max "ibp-geodns/src/common/maxmind"
)

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
