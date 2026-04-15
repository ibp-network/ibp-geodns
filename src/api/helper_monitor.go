package api

import (
	"strings"
)

func offSite(sites []MonitorResultSite, member string, v6 *bool) bool {
	for _, sr := range sites {
		if v6 != nil && sr.IsIPv6 != *v6 {
			continue
		}
		for _, r := range sr.Results {
			if strings.EqualFold(r.MemberName, member) && !r.Status {
				return true
			}
		}
	}
	return false
}

func offDomain(domains []MonitorResultDomain, member, dom string, v6 *bool) bool {
	for _, dr := range domains {
		if !strings.EqualFold(dr.Domain, dom) {
			continue
		}
		if v6 != nil && dr.IsIPv6 != *v6 {
			continue
		}
		for _, r := range dr.Results {
			if strings.EqualFold(r.MemberName, member) && !r.Status {
				return true
			}
		}
	}
	return false
}

func offEndpoint(eps []MonitorResultEndpoint, member, dom string, v6 *bool) bool {
	for _, er := range eps {
		if !strings.EqualFold(er.Domain, dom) {
			continue
		}
		if v6 != nil && er.IsIPv6 != *v6 {
			continue
		}
		for _, r := range er.Results {
			if strings.EqualFold(r.MemberName, member) && !r.Status {
				return true
			}
		}
	}
	return false
}

func shouldApplyMonitorSnapshot(s OfficialResults) bool {
	return HasOfficialSnapshot() && HasOfficialMonitorData(s)
}

func IsMemberOnlineForDomain(domain, member string) bool {
	s := GetOfficialSnapshot()
	if !shouldApplyMonitorSnapshot(s) {
		return true
	}
	if offSite(s.SiteResults, member, nil) ||
		offDomain(s.DomainResults, member, domain, nil) ||
		offEndpoint(s.EndpointResults, member, domain, nil) {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv4(domain, member string) bool {
	ipv6 := false
	s := GetOfficialSnapshot()
	if !shouldApplyMonitorSnapshot(s) {
		return true
	}
	if offSite(s.SiteResults, member, &ipv6) ||
		offDomain(s.DomainResults, member, domain, &ipv6) ||
		offEndpoint(s.EndpointResults, member, domain, &ipv6) {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv6(domain, member string) bool {
	ipv6 := true
	s := GetOfficialSnapshot()
	if !shouldApplyMonitorSnapshot(s) {
		return true
	}
	if offSite(s.SiteResults, member, &ipv6) ||
		offDomain(s.DomainResults, member, domain, &ipv6) ||
		offEndpoint(s.EndpointResults, member, domain, &ipv6) {
		return false
	}
	return true
}
