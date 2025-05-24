package api

import (
	"strings"

	// We use the updated snapshot from dnsApi's global officialResultsSnapshot
	// but we need a reference to it. We'll import from main if needed.

	. "ibp-geodns/src/dnsApi"
)

// for officialResultsSnapshot, resultsMu

// IsMemberOnlineForDomain checks the officialResultsSnapshot for a given member & domain
func IsMemberOnlineForDomain(domain, memberName string) bool {
	resultsMu.RLock()
	defer resultsMu.RUnlock()

	sites := officialResultsSnapshot.SiteResults
	domains := officialResultsSnapshot.DomainResults
	endpoints := officialResultsSnapshot.EndpointResults

	// Check site-level
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.MemberName == memberName && !r.Status {
				return false
			}
		}
	}

	// Check domain-level
	for _, dr := range domains {
		if strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.MemberName == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check endpoint-level
	for _, er := range endpoints {
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
