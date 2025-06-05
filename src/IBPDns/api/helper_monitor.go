package api

import "strings"

// IsMemberOnlineForDomain checks the *official* snapshot for a member's site/domain/endpoint status
// A member is considered OFFLINE if they fail ANY relevant check (IPv4 OR IPv6, site OR domain OR endpoint)
func IsMemberOnlineForDomain(domain, memberName string) bool {
	// Retrieve the official results snapshot
	snap := GetOfficialSnapshot()

	// Check site-level failures (both IPv4 and IPv6)
	// If ANY site check fails, member is offline
	for _, sr := range snap.SiteResults {
		for _, r := range sr.Results {
			if r.MemberName == memberName && !r.Status {
				return false // Failed site check = offline
			}
		}
	}

	// Check domain-level failures for this specific domain (both IPv4 and IPv6)
	// If ANY domain check fails for this domain, member is offline for this domain
	for _, dr := range snap.DomainResults {
		if strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.MemberName == memberName && !r.Status {
					return false // Failed domain check = offline for this domain
				}
			}
		}
	}

	// Check endpoint-level failures for this specific domain (both IPv4 and IPv6)
	// If ANY endpoint check fails for this domain, member is offline for this domain
	for _, er := range snap.EndpointResults {
		if strings.EqualFold(er.Domain, domain) {
			for _, r := range er.Results {
				if r.MemberName == memberName && !r.Status {
					return false // Failed endpoint check = offline for this domain
				}
			}
		}
	}

	return true // No failures found = online
}

// IsMemberOnlineForDomainIPv4 checks only IPv4-related status for a domain
// This is used when we specifically want to serve IPv4 records
func IsMemberOnlineForDomainIPv4(domain, memberName string) bool {
	snap := GetOfficialSnapshot()

	// Check IPv4 site-level failures
	for _, sr := range snap.SiteResults {
		if !sr.IsIPv6 { // Only check IPv4 site results
			for _, r := range sr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check IPv4 domain-level failures for this specific domain
	for _, dr := range snap.DomainResults {
		if !dr.IsIPv6 && strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check IPv4 endpoint-level failures for this specific domain
	for _, er := range snap.EndpointResults {
		if !er.IsIPv6 && strings.EqualFold(er.Domain, domain) {
			for _, r := range er.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	return true
}

// IsMemberOnlineForDomainIPv6 checks only IPv6-related status for a domain
// This is used when we specifically want to serve IPv6 records
func IsMemberOnlineForDomainIPv6(domain, memberName string) bool {
	snap := GetOfficialSnapshot()

	// Check IPv6 site-level failures
	for _, sr := range snap.SiteResults {
		if sr.IsIPv6 { // Only check IPv6 site results
			for _, r := range sr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check IPv6 domain-level failures for this specific domain
	for _, dr := range snap.DomainResults {
		if dr.IsIPv6 && strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check IPv6 endpoint-level failures for this specific domain
	for _, er := range snap.EndpointResults {
		if er.IsIPv6 && strings.EqualFold(er.Domain, domain) {
			for _, r := range er.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	return true
}
