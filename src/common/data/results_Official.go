package data

import (
	"common/config"
	"sync"
	"time"
)

/*
 *
 *  Functions for storing and handling official results & status
 *
 */

var Official = OfficialResults{
	SiteResults:     make([]SiteResult, 0),
	DomainResults:   make([]DomainResult, 0),
	EndpointResults: make([]EndpointResult, 0),
	Mu:              sync.RWMutex{},
}

// Official Results Functions
func SetOfficialSiteResults(results []SiteResult) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()
	Official.SiteResults = results
}

func SetOfficialDomainResults(results []DomainResult) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()
	Official.DomainResults = results
}

func SetOfficialEndpointResults(results []EndpointResult) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()
	Official.EndpointResults = results
}

func GetOfficialResults() (sites []SiteResult, domains []DomainResult, endpoints []EndpointResult) {
	Official.Mu.RLock()
	defer Official.Mu.RUnlock()
	return Official.SiteResults, Official.DomainResults, Official.EndpointResults
}

// GetOfficialSiteStatus returns the status of a specific site check for a given member.
func GetOfficialSiteStatus(checkName string, memberName string) (found bool, status bool) {
	officialSites, _, _ := GetOfficialResults()
	for _, osr := range officialSites {
		if osr.Check.Name == checkName {
			for _, r := range osr.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
			break
		}
	}
	return false, false
}

// GetOfficialDomainStatus returns the status of a specific domain check for a given member and domain.
func GetOfficialDomainStatus(checkName string, memberName string, domain string) (found bool, status bool) {
	_, officialDomains, _ := GetOfficialResults()
	for _, od := range officialDomains {
		if od.Check.Name == checkName && od.Domain == domain {
			for _, r := range od.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
			break
		}
	}
	return false, false
}

// GetOfficialEndpointStatus returns the status of a specific endpoint check for a given member, domain, and endpoint.
func GetOfficialEndpointStatus(checkName string, memberName string, domain string, endpoint string) (found bool, status bool) {
	_, _, officialEndpoints := GetOfficialResults()
	for _, oe := range officialEndpoints {
		if oe.Check.Name == checkName && oe.RpcUrl == endpoint && oe.Domain == domain {
			for _, r := range oe.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
			break
		}
	}
	return false, false
}

// UpdateOfficialSiteResult updates the official site results for a given check and member.
func UpdateOfficialSiteResult(check config.Check, member config.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	sIndex := -1
	for i, sr := range Official.SiteResults {
		if sr.Check.Name == check.Name {
			sIndex = i
			break
		}
	}

	newResult := Result{
		Member:    member,
		Status:    status,
		Checktime: time.Now().UTC(),
		ErrorText: errorMsg,
		Data:      dataMap,
	}

	if sIndex == -1 {
		// Create a new site entry
		Official.SiteResults = append(Official.SiteResults, SiteResult{
			Check:   check,
			Results: []Result{newResult},
		})

		// If the site has been offline since startup, record an offline event
		if !status {
			go RecordEvent("site", check.Name, member.Details.Name, "", "", false, "Offline since startup", dataMap)
		}
	} else {
		sr := &Official.SiteResults[sIndex]

		rIndex := -1
		for i, res := range sr.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			sr.Results = append(sr.Results, newResult)
			// If the site has been offline since startup, record an offline event
			if !status {
				go RecordEvent("site", check.Name, member.Details.Name, "", "", false, "Offline since startup", dataMap)
			}
		} else {
			// Log the change if there's a status difference
			if sr.Results[rIndex].Status != status {
				go RecordEvent("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap)
			}
			sr.Results[rIndex] = newResult
		}
	}
}

// UpdateOfficialDomainResult updates the official domain results for a given check, domain, service, and member.
func UpdateOfficialDomainResult(check config.Check, member config.Member, service config.Service, domain string, status bool, errorMsg string, dataMap map[string]interface{}) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	dIndex := -1
	for i, dr := range Official.DomainResults {
		if dr.Check.Name == check.Name && dr.Domain == domain {
			dIndex = i
			break
		}
	}

	newResult := Result{
		Member:    member,
		Status:    status,
		Checktime: time.Now().UTC(),
		ErrorText: errorMsg,
		Data:      dataMap,
	}

	if dIndex == -1 {
		// Create a new domain entry
		Official.DomainResults = append(Official.DomainResults, DomainResult{
			Check:   check,
			Service: service,
			Domain:  domain,
			Results: []Result{newResult},
		})

		// If the domain has been offline since startup, record an offline event
		if !status {
			go RecordEvent("domain", check.Name, member.Details.Name, domain, "", false, "Offline since startup", dataMap)
		}
	} else {
		dr := &Official.DomainResults[dIndex]

		rIndex := -1
		for i, res := range dr.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			dr.Results = append(dr.Results, newResult)
			// If the domain has been offline since startup, record an offline event
			if !status {
				go RecordEvent("domain", check.Name, member.Details.Name, domain, "", false, "Offline since startup", dataMap)
			}
		} else {
			// Log the change if there's a status difference
			if dr.Results[rIndex].Status != status {
				go RecordEvent("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap)
			}
			dr.Results[rIndex] = newResult
		}
	}
}

// UpdateOfficialEndpointResult updates the official endpoint results for a given check, endpoint, service, and member.
func UpdateOfficialEndpointResult(check config.Check, member config.Member, service config.Service, domain string, endpoint string, status bool, errorMsg string, dataMap map[string]interface{}) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	eIndex := -1
	for i, er := range Official.EndpointResults {
		if er.Check.Name == check.Name && er.Domain == domain && er.RpcUrl == endpoint {
			eIndex = i
			break
		}
	}

	newResult := Result{
		Member:    member,
		Status:    status,
		Checktime: time.Now().UTC(),
		ErrorText: errorMsg,
		Data:      dataMap,
	}

	if eIndex == -1 {
		// Create a new endpoint entry
		Official.EndpointResults = append(Official.EndpointResults, EndpointResult{
			Check:   check,
			Service: service,
			RpcUrl:  endpoint,
			Domain:  domain,
			Results: []Result{newResult},
		})

		// If the endpoint has been offline since startup, record an offline event
		if !status {
			go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, false, "Offline since startup", dataMap)
		}
	} else {
		er := &Official.EndpointResults[eIndex]

		rIndex := -1
		for i, res := range er.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			er.Results = append(er.Results, newResult)
			// If the endpoint has been offline since startup, record an offline event
			if !status {
				go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, false, "Offline since startup", dataMap)
			}
		} else {
			// Log the change if there's a status difference
			if er.Results[rIndex].Status != status {
				go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, status, errorMsg, dataMap)
			}
			er.Results[rIndex] = newResult
		}
	}
}
