package api

import "strings"

// IsMemberOnlineForDomain checks the local snapshot for a member's status on site/domain/endpoint
func IsMemberOnlineForDomain(domain, memberName string) bool {
	// Grab a copy of the local snapshot
	snap := GetLocalSnapshot()

	// Check all site-level results
	for _, sr := range snap.SiteResults {
		for _, r := range sr.Results {
			if r.MemberName == memberName && !r.Status {
				return false
			}
		}
	}

	// Check domain-level
	for _, dr := range snap.DomainResults {
		if strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check endpoint-level
	for _, er := range snap.EndpointResults {
		if strings.EqualFold(er.Domain, domain) {
			for _, r := range er.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}
	return true
}
