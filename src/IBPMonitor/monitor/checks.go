package monitor

import (
	"net/url"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	natsCommon "ibp-geodns/src/common/nats"
)

var CheckRegistry = struct {
	Site     map[string]CheckSiteFunc
	Domain   map[string]CheckDomainFunc
	Endpoint map[string]CheckEndpointFunc
}{
	Site:     make(map[string]CheckSiteFunc),
	Domain:   make(map[string]CheckDomainFunc),
	Endpoint: make(map[string]CheckEndpointFunc),
}

type (
	CheckSiteFunc     func(check cfg.Check, member cfg.Member)
	CheckDomainFunc   func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)
	CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)
)

// ServiceTypeValidator holds service type validation info for checks
var ServiceTypeValidator = struct {
	Domain   map[string][]string
	Endpoint map[string][]string
}{
	Domain:   make(map[string][]string),
	Endpoint: make(map[string][]string),
}

func RegisterSiteCheck(name string, fn CheckSiteFunc) {
	CheckRegistry.Site[name] = fn
}

func RegisterDomainCheck(name string, fn CheckDomainFunc) {
	CheckRegistry.Domain[name] = fn
}

func RegisterDomainCheckWithTypes(name string, fn CheckDomainFunc, validTypes []string) {
	CheckRegistry.Domain[name] = fn
	ServiceTypeValidator.Domain[name] = validTypes
}

func RegisterEndpointCheck(name string, fn CheckEndpointFunc) {
	CheckRegistry.Endpoint[name] = fn
}

func RegisterEndpointCheckWithTypes(name string, fn CheckEndpointFunc, validTypes []string) {
	CheckRegistry.Endpoint[name] = fn
	ServiceTypeValidator.Endpoint[name] = validTypes
}

func isCheckValidForServiceType(checkName string, checkType string, serviceType string) bool {
	var validTypes []string

	switch checkType {
	case "domain":
		validTypes = ServiceTypeValidator.Domain[checkName]
	case "endpoint":
		validTypes = ServiceTypeValidator.Endpoint[checkName]
	default:
		// Site checks don't have service type restrictions
		return true
	}

	// If no valid types specified, allow all
	if len(validTypes) == 0 {
		return true
	}

	// Check if service type is in valid types list
	for _, vt := range validTypes {
		if strings.EqualFold(vt, serviceType) {
			return true
		}
	}

	return false
}

func startChecks() {
	go initSiteCheck()
	go initDomainCheck()
	go initEndpointCheck()
}

func getSiteCheck(name string) (CheckSiteFunc, bool) {
	fn, ok := CheckRegistry.Site[name]
	return fn, ok
}

func initSiteCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "site" && ch.Enabled == 1 {
			if fn, ok := getSiteCheck(ch.Name); ok {
				go siteCheckTimer(ch, fn)
			}
		}
	}
}

func siteCheckTimer(ch cfg.Check, fn CheckSiteFunc) {
	time.Sleep(2 * time.Second)
	runSiteCheck(ch, fn)
	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		runSiteCheck(ch, fn)
	}
}

func runSiteCheck(ch cfg.Check, fn CheckSiteFunc) {
	c := cfg.GetConfig()
	for _, m := range c.Members {
		if m.Service.Active == 1 && !m.Override {
			time.Sleep(25 * time.Millisecond)
			go func(check cfg.Check, mem cfg.Member) {
				defer func() {
					if r := recover(); r != nil {
						log.Log(log.Error, "SITE-check %s on %s panicked: %v", check.Name, mem.Details.Name, r)
					}
				}()
				fn(check, mem)
			}(ch, m)
		}
	}
}

func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errText string,
	data map[string]interface{}, ipv6 bool) {
	dat.UpdateLocalSiteResult(check, member, status, errText, data, ipv6)
	proposeIfStatusChanged("site", check.Name, member.Details.Name, "", "",
		status, errText, data, ipv6)
}

func getDomainCheck(name string) (CheckDomainFunc, bool) {
	fn, ok := CheckRegistry.Domain[name]
	return fn, ok
}

func initDomainCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "domain" && ch.Enabled == 1 {
			if fn, ok := getDomainCheck(ch.Name); ok {
				go domainCheckTimer(ch, fn)
			}
		}
	}
}

func domainCheckTimer(ch cfg.Check, fn CheckDomainFunc) {
	time.Sleep(3 * time.Second)
	runDomainCheck(ch, fn)
	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		runDomainCheck(ch, fn)
	}
}

func runDomainCheck(ch cfg.Check, fn CheckDomainFunc) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		// Check if this check is valid for this service type
		if !isCheckValidForServiceType(ch.Name, "domain", svc.Configuration.ServiceType) {
			continue
		}

		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired {
				if assignedToService(svcName, mem) {
					doms := extractDomains(svc)
					for dom := range doms {
						go func(check cfg.Check, domain string, s cfg.Service, m cfg.Member) {
							defer func() {
								if r := recover(); r != nil {
									log.Log(log.Error, "DOMAIN-check %s on %s crashed: %v", check.Name, m.Details.Name, r)
								}
							}()
							fn(check, domain, s, m)
						}(ch, dom, svc, mem)
						time.Sleep(25 * time.Millisecond)
					}
				}
			}
		}
	}
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service,
	member cfg.Member, status bool, errText string, data map[string]interface{}, ipv6 bool) {
	dat.UpdateLocalDomainResult(check, member, service, domain, status, errText, data, ipv6)
	proposeIfStatusChanged("domain", check.Name, member.Details.Name, domain, "",
		status, errText, data, ipv6)
}

func getEndpointCheck(name string) (CheckEndpointFunc, bool) {
	fn, ok := CheckRegistry.Endpoint[name]
	return fn, ok
}

func initEndpointCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "endpoint" && ch.Enabled == 1 {
			if fn, ok := getEndpointCheck(ch.Name); ok {
				go endpointCheckTimer(ch, fn)
			}
		}
	}
}

func endpointCheckTimer(ch cfg.Check, fn CheckEndpointFunc) {
	time.Sleep(4 * time.Second)
	runEndpointCheck(ch, fn)
	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		runEndpointCheck(ch, fn)
	}
}

func runEndpointCheck(ch cfg.Check, fn CheckEndpointFunc) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		// Check if this check is valid for this service type
		if !isCheckValidForServiceType(ch.Name, "endpoint", svc.Configuration.ServiceType) {
			continue
		}

		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired {
				if assignedToService(svcName, mem) {
					for _, prov := range svc.Providers {
						for _, rpc := range prov.RpcUrls {
							go func(check cfg.Check, endpoint string, s cfg.Service, m cfg.Member) {
								defer func() {
									if r := recover(); r != nil {
										log.Log(log.Error, "ENDPOINT-check %s on %s crashed: %v", check.Name, m.Details.Name, r)
									}
								}()
								fn(check, endpoint, s, m)
							}(ch, rpc, svc, mem)
							time.Sleep(25 * time.Millisecond)
						}
					}
				}
			}
		}
	}
}

func UpdateEndpointResultLocal(check cfg.Check, member cfg.Member, service cfg.Service,
	endpoint string, status bool, errText string, data map[string]interface{}, ipv6 bool) {
	domain := parseUrlForDomain(endpoint)
	dat.UpdateLocalEndpointResult(check, member, service, domain, endpoint, status, errText, data, ipv6)
	proposeIfStatusChanged("endpoint", check.Name, member.Details.Name, domain, endpoint,
		status, errText, data, ipv6)
}

func proposeIfStatusChanged(checkType, checkName, memberName, domainName, endpoint string,
	status bool, errText string, data map[string]interface{}, ipv6 bool) {
	var (
		found bool
		cur   bool
	)
	switch checkType {
	case "site":
		found, cur = dat.GetOfficialSiteStatus(checkName, memberName, ipv6)
	case "domain":
		found, cur = dat.GetOfficialDomainStatus(checkName, memberName, domainName, ipv6)
	case "endpoint":
		found, cur = dat.GetOfficialEndpointStatus(checkName, memberName, domainName, endpoint, ipv6)
	}
	if !found || cur != status {
		natsCommon.ProposeCheckStatus(
			checkType,
			checkName,
			memberName,
			domainName,
			endpoint,
			status,
			errText,
			data,
			ipv6,
		)
	}
}

func assignedToService(svcName string, m cfg.Member) bool {
	for _, list := range m.ServiceAssignments {
		for _, v := range list {
			if v == svcName {
				return true
			}
		}
	}
	return false
}

func extractDomains(s cfg.Service) map[string]struct{} {
	out := make(map[string]struct{})
	for _, prov := range s.Providers {
		for _, rpc := range prov.RpcUrls {
			if d := parseUrlForDomain(rpc); d != "" {
				out[d] = struct{}{}
			}
		}
	}
	return out
}

func parseUrlForDomain(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
