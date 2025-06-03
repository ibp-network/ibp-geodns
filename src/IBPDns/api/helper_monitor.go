package api

import (
	"strings"
)

// IsMemberOnlineForDomain checks the *official* snapshot for a member’s site/domain/endpoint status
func IsMemberOnlineForDomain(domain, memberName string) bool {
	// Retrieve the official results snapshot
	snap := GetOfficialSnapshot()

	// Check site-level
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
