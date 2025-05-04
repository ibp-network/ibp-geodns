package data

import (
	"ibp-geodns/config"
	"sync"
	"time"
)

/*
 *
 *  Functions for storing and handling site results & status
 *
 */

var Local = LocalResults{
	SiteResults:     make([]SiteResult, 0),
	DomainResults:   make([]DomainResult, 0),
	EndpointResults: make([]EndpointResult, 0),
	Mu:              sync.RWMutex{},
}

// Local Results Functions
func SetLocalSiteResults(results []SiteResult) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()
	Local.SiteResults = results
}

func SetLocalDomainResults(results []DomainResult) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()
	Local.DomainResults = results
}

func SetLocalEndpointResults(results []EndpointResult) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()
	Local.EndpointResults = results
}

func GetLocalResults() (sites []SiteResult, domains []DomainResult, endpoints []EndpointResult) {
	Local.Mu.RLock()
	defer Local.Mu.RUnlock()
	return Local.SiteResults, Local.DomainResults, Local.EndpointResults
}

// Update Local Results
func UpdateLocalSiteResult(check config.Check, member config.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	sIndex := -1
	for i, sr := range Local.SiteResults {
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
		Local.SiteResults = append(Local.SiteResults, SiteResult{
			Check:   check,
			Results: []Result{newResult},
		})
	} else {
		sr := &Local.SiteResults[sIndex]

		rIndex := -1
		for i, res := range sr.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			sr.Results = append(sr.Results, newResult)
		} else {
			sr.Results[rIndex] = newResult
		}
	}
}

func UpdateLocalDomainResult(check config.Check, member config.Member, service config.Service, domain string, status bool, errorMsg string, dataMap map[string]interface{}) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	dIndex := -1
	for i, dr := range Local.DomainResults {
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
		Local.DomainResults = append(Local.DomainResults, DomainResult{
			Check:   check,
			Service: service,
			Domain:  domain,
			Results: []Result{newResult},
		})
	} else {
		dr := &Local.DomainResults[dIndex]

		rIndex := -1
		for i, res := range dr.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			dr.Results = append(dr.Results, newResult)
		} else {
			dr.Results[rIndex] = newResult
		}
	}
}

func UpdateLocalEndpointResult(check config.Check, member config.Member, service config.Service, domain string, endpoint string, status bool, errorMsg string, dataMap map[string]interface{}) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	eIndex := -1
	for i, er := range Local.EndpointResults {
		if er.Check.Name == check.Name && er.RpcUrl == endpoint {
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
		Local.EndpointResults = append(Local.EndpointResults, EndpointResult{
			Check:   check,
			Service: service,
			RpcUrl:  endpoint,
			Domain:  domain,
			Results: []Result{newResult},
		})
	} else {
		er := &Local.EndpointResults[eIndex]

		rIndex := -1
		for i, res := range er.Results {
			if res.Member.Details.Name == member.Details.Name {
				rIndex = i
				break
			}
		}
		if rIndex == -1 {
			er.Results = append(er.Results, newResult)
		} else {
			er.Results[rIndex] = newResult
		}
	}
}

// Local Status Helpers
func GetLocalSiteStatus(checkName string, memberName string) (found bool, status bool) {
	localSites, _, _ := GetLocalResults()
	for _, lsr := range localSites {
		if lsr.Check.Name == checkName {
			for _, r := range lsr.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
			break
		}
	}
	return false, false
}

func GetLocalDomainStatus(checkName string, memberName string, domain string) (found bool, status bool) {
	_, localDomains, _ := GetLocalResults()
	for _, ld := range localDomains {
		if ld.Check.Name == checkName && ld.Domain == domain {
			for _, r := range ld.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}

			break
		}
	}
	return false, false
}

func GetLocalEndpointStatus(checkName string, memberName string, endpoint string) (found bool, status bool) {
	_, _, localEndpoints := GetLocalResults()
	for _, le := range localEndpoints {
		if le.Check.Name == checkName && le.RpcUrl == endpoint {
			for _, r := range le.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}

			break
		}
	}
	return false, false
}
