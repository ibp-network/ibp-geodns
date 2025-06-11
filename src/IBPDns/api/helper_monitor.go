package api

import (
	"strings"
)

/*
   This file now works with **offline‑only** snapshots.

   If any matching record is present ⇒ the member is OFFLINE.
   If no record exists           ⇒ the member is ONLINE.
*/

// --------------- helpers -----------------------------------------------------
func isOfflineSite(sites []MonitorResultSite, member string, v6Filter *bool) bool {
	for _, sr := range sites {
		if v6Filter != nil && sr.IsIPv6 != *v6Filter {
			continue
		}
		for _, r := range sr.Results {
			if r.MemberName == member {
				return true
			}
		}
	}
	return false
}

func isOfflineDomain(domains []MonitorResultDomain, member, domain string, v6Filter *bool) bool {
	for _, dr := range domains {
		if !strings.EqualFold(dr.Domain, domain) {
			continue
		}
		if v6Filter != nil && dr.IsIPv6 != *v6Filter {
			continue
		}
		for _, r := range dr.Results {
			if r.MemberName == member {
				return true
			}
		}
	}
	return false
}

func isOfflineEndpoint(eps []MonitorResultEndpoint, member, domain string, v6Filter *bool) bool {
	for _, er := range eps {
		if !strings.EqualFold(er.Domain, domain) {
			continue
		}
		if v6Filter != nil && er.IsIPv6 != *v6Filter {
			continue
		}
		for _, r := range er.Results {
			if r.MemberName == member {
				return true
			}
		}
	}
	return false
}

// --------------- public ------------------------------------------------------

// ONLINE if **no** corresponding offline record exists
func IsMemberOnlineForDomain(domain, member string) bool {
	snap := GetOfficialSnapshot()
	if isOfflineSite(snap.SiteResults, member, nil) ||
		isOfflineDomain(snap.DomainResults, member, domain, nil) ||
		isOfflineEndpoint(snap.EndpointResults, member, domain, nil) {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv4(domain, member string) bool {
	ipv6 := false
	snap := GetOfficialSnapshot()
	if isOfflineSite(snap.SiteResults, member, &ipv6) ||
		isOfflineDomain(snap.DomainResults, member, domain, &ipv6) ||
		isOfflineEndpoint(snap.EndpointResults, member, domain, &ipv6) {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv6(domain, member string) bool {
	ipv6 := true
	snap := GetOfficialSnapshot()
	if isOfflineSite(snap.SiteResults, member, &ipv6) ||
		isOfflineDomain(snap.DomainResults, member, domain, &ipv6) ||
		isOfflineEndpoint(snap.EndpointResults, member, domain, &ipv6) {
		return false
	}
	return true
}
