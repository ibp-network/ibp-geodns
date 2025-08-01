package monitor

import (
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	natsCommon "ibp-geodns/src/common/nats"
	"net/url"
	"strings"
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

func getSiteCheck(name string) (CheckSiteFunc, bool) {
	fn, ok := CheckRegistry.Site[name]
	return fn, ok
}

func getDomainCheck(name string) (CheckDomainFunc, bool) {
	fn, ok := CheckRegistry.Domain[name]
	return fn, ok
}

func getEndpointCheck(name string) (CheckEndpointFunc, bool) {
	fn, ok := CheckRegistry.Endpoint[name]
	return fn, ok
}

func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errText string,
	data map[string]interface{}, ipv6 bool) {
	dat.UpdateLocalSiteResult(check, member, status, errText, data, ipv6)
	proposeIfStatusChanged("site", check.Name, member.Details.Name, "", "",
		status, errText, data, ipv6)
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service,
	member cfg.Member, status bool, errText string, data map[string]interface{}, ipv6 bool) {
	dat.UpdateLocalDomainResult(check, member, service, domain, status, errText, data, ipv6)
	proposeIfStatusChanged("domain", check.Name, member.Details.Name, domain, "",
		status, errText, data, ipv6)
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
