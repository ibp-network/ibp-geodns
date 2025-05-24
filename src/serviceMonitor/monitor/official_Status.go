package monitor

import "strings"

func getOfficialSiteStatus(checkName, memberName string) (bool, bool) {
	officialMu.RLock()
	defer officialMu.RUnlock()

	for _, sr := range official.SiteResults {
		if sr.Check.Name == checkName {
			for _, r := range sr.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}

func getOfficialDomainStatus(checkName, memberName, domain string) (bool, bool) {
	officialMu.RLock()
	defer officialMu.RUnlock()

	for _, dr := range official.DomainResults {
		if dr.Check.Name == checkName && strings.EqualFold(dr.Domain, domain) {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}

func getOfficialEndpointStatus(checkName, memberName, domain, endpoint string) (bool, bool) {
	officialMu.RLock()
	defer officialMu.RUnlock()

	for _, er := range official.EndpointResults {
		if er.Check.Name == checkName && strings.EqualFold(er.Domain, domain) && er.RpcUrl == endpoint {
			for _, r := range er.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}
