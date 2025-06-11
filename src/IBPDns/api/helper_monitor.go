package api

import (
	"strings"
	"time"
)

func latestStatus(results []MonitorResultGeneric, member string, ipv6Filter *bool) (bool, bool) {
	found := false
	var newest time.Time
	var latest bool
	for _, r := range results {
		if r.MemberName != member {
			continue
		}
		if ipv6Filter != nil && r.IsIPv6 != *ipv6Filter {
			continue
		}
		if !found || r.Checktime.After(newest) {
			found = true
			latest = r.Status
			newest = r.Checktime
		}
	}
	return found, latest
}

/* ---------------- Aggregate helpers ---------------------------*/

func newestSiteStatus(sites []MonitorResultSite, member string, ipv6Filter *bool) (bool, bool) {
	found := false
	var newest time.Time
	var latest bool
	for _, sr := range sites {
		if ipv6Filter != nil && sr.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st := latestStatus(sr.Results, member, ipv6Filter); ok {
			if !found || moreRecent(sr.Results, member, ipv6Filter, newest) {
				found, latest = true, st
				newest = newestChecktime(sr.Results, member, ipv6Filter)
			}
		}
	}
	return found, latest
}

func newestDomainStatus(domains []MonitorResultDomain, member, domain string, ipv6Filter *bool) (bool, bool) {
	found := false
	var newest time.Time
	var latest bool
	for _, dr := range domains {
		if !strings.EqualFold(dr.Domain, domain) {
			continue
		}
		if ipv6Filter != nil && dr.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st := latestStatus(dr.Results, member, ipv6Filter); ok {
			if !found || moreRecent(dr.Results, member, ipv6Filter, newest) {
				found, latest = true, st
				newest = newestChecktime(dr.Results, member, ipv6Filter)
			}
		}
	}
	return found, latest
}

func newestEndpointStatus(endpoints []MonitorResultEndpoint, member, domain string, ipv6Filter *bool) (bool, bool) {
	found := false
	var newest time.Time
	var latest bool
	for _, er := range endpoints {
		if !strings.EqualFold(er.Domain, domain) {
			continue
		}
		if ipv6Filter != nil && er.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st := latestStatus(er.Results, member, ipv6Filter); ok {
			if !found || moreRecent(er.Results, member, ipv6Filter, newest) {
				found, latest = true, st
				newest = newestChecktime(er.Results, member, ipv6Filter)
			}
		}
	}
	return found, latest
}

func newestChecktime(results []MonitorResultGeneric, member string, ipv6Filter *bool) time.Time {
	var newest time.Time
	for _, r := range results {
		if r.MemberName == member && (ipv6Filter == nil || r.IsIPv6 == *ipv6Filter) {
			if r.Checktime.After(newest) {
				newest = r.Checktime
			}
		}
	}
	return newest
}

func moreRecent(results []MonitorResultGeneric, member string, ipv6Filter *bool, ref time.Time) bool {
	for _, r := range results {
		if r.MemberName == member && (ipv6Filter == nil || r.IsIPv6 == *ipv6Filter) {
			if r.Checktime.After(ref) {
				return true
			}
		}
	}
	return false
}

/* ---------------- Public helpers ------------------------------*/

func IsMemberOnlineForDomain(domain, member string) bool {
	sites, domains, eps := GetOfficialSnapshot().SiteResults, GetOfficialSnapshot().DomainResults, GetOfficialSnapshot().EndpointResults

	if ok, st := newestSiteStatus(sites, member, nil); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, member, domain, nil); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(eps, member, domain, nil); ok && !st {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv4(domain, member string) bool {
	ipv6 := false
	sites, domains, eps := GetOfficialSnapshot().SiteResults, GetOfficialSnapshot().DomainResults, GetOfficialSnapshot().EndpointResults

	if ok, st := newestSiteStatus(sites, member, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, member, domain, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(eps, member, domain, &ipv6); ok && !st {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv6(domain, member string) bool {
	ipv6 := true
	sites, domains, eps := GetOfficialSnapshot().SiteResults, GetOfficialSnapshot().DomainResults, GetOfficialSnapshot().EndpointResults

	if ok, st := newestSiteStatus(sites, member, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, member, domain, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(eps, member, domain, &ipv6); ok && !st {
		return false
	}
	return true
}
