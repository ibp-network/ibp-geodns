package data

import (
	cfg "ibp-geodns/src/common/config"
	"sync"
	"time"
)

var Local = LocalResults{
	SiteResults:     make([]SiteResult, 0),
	DomainResults:   make([]DomainResult, 0),
	EndpointResults: make([]EndpointResult, 0),
	Mu:              sync.RWMutex{},
}

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

func UpdateLocalSiteResult(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {
	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	sIndex := -1
	for i, sr := range Local.SiteResults {
		if sr.Check.Name == check.Name && sr.IsIPv6 == isIPv6 {
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
		IsIPv6:    isIPv6,
	}

	if sIndex == -1 {
		site := SiteResult{
			Check:   check,
			IsIPv6:  isIPv6,
			Results: []Result{newResult},
		}
		Local.SiteResults = append(Local.SiteResults, site)
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

func UpdateLocalDomainResult(check cfg.Check, member cfg.Member, service cfg.Service, domain string,
	status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {

	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	dIndex := -1
	for i, dr := range Local.DomainResults {
		if dr.Check.Name == check.Name && dr.Domain == domain && dr.IsIPv6 == isIPv6 {
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
		IsIPv6:    isIPv6,
	}

	if dIndex == -1 {
		Local.DomainResults = append(Local.DomainResults, DomainResult{
			Check:   check,
			Service: service,
			Domain:  domain,
			IsIPv6:  isIPv6,
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

func UpdateLocalEndpointResult(check cfg.Check, member cfg.Member, service cfg.Service, domain string, endpoint string,
	status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {

	Local.Mu.Lock()
	defer Local.Mu.Unlock()

	eIndex := -1
	for i, er := range Local.EndpointResults {
		if er.Check.Name == check.Name && er.Domain == domain && er.RpcUrl == endpoint && er.IsIPv6 == isIPv6 {
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
		IsIPv6:    isIPv6,
	}

	if eIndex == -1 {
		Local.EndpointResults = append(Local.EndpointResults, EndpointResult{
			Check:   check,
			Service: service,
			RpcUrl:  endpoint,
			Domain:  domain,
			IsIPv6:  isIPv6,
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

func GetLocalSiteStatusIPv4v6(checkName, memberName string, isIPv6 bool) (bool, bool) {
	Local.Mu.RLock()
	defer Local.Mu.RUnlock()
	for _, lsr := range Local.SiteResults {
		if lsr.Check.Name == checkName && lsr.IsIPv6 == isIPv6 {
			for _, r := range lsr.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}

func GetLocalDomainStatusIPv4v6(checkName, memberName, domain string, isIPv6 bool) (bool, bool) {
	Local.Mu.RLock()
	defer Local.Mu.RUnlock()
	for _, ld := range Local.DomainResults {
		if ld.Check.Name == checkName && ld.Domain == domain && ld.IsIPv6 == isIPv6 {
			for _, r := range ld.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}

func GetLocalEndpointStatusIPv4v6(checkName, memberName, domain, endpoint string, isIPv6 bool) (bool, bool) {
	Local.Mu.RLock()
	defer Local.Mu.RUnlock()
	for _, le := range Local.EndpointResults {
		if le.Check.Name == checkName && le.Domain == domain && le.RpcUrl == endpoint && le.IsIPv6 == isIPv6 {
			for _, r := range le.Results {
				if r.Member.Details.Name == memberName {
					return true, r.Status
				}
			}
		}
	}
	return false, false
}
