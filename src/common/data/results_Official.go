package data

import (
	cfg "ibp-geodns/src/common/config"
	"sync"
	"time"
)

var Official = OfficialResults{
	SiteResults:     make([]SiteResult, 0),
	DomainResults:   make([]DomainResult, 0),
	EndpointResults: make([]EndpointResult, 0),
	Mu:              sync.RWMutex{},
}

type Snapshot struct {
	SiteResults     []SiteResult     `json:"site"`
	DomainResults   []DomainResult   `json:"domain"`
	EndpointResults []EndpointResult `json:"endpoint"`
}

var (
	muOfficial sync.RWMutex
	official   Snapshot
)

func GetOfficialResults() ([]SiteResult, []DomainResult, []EndpointResult) {
	muOfficial.RLock()
	defer muOfficial.RUnlock()
	return official.SiteResults, official.DomainResults, official.EndpointResults
}

func SetOfficialSnapshot(snap Snapshot) {
	muOfficial.Lock()
	official = snap
	muOfficial.Unlock()
}

func BuildSnapshot(site []SiteResult, dom []DomainResult, eps []EndpointResult) Snapshot {
	return Snapshot{
		SiteResults:     site,
		DomainResults:   dom,
		EndpointResults: eps,
	}
}

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

func UpdateOfficialSiteResult(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {
	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	sIndex := -1
	for i, sr := range Official.SiteResults {
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
		Official.SiteResults = append(Official.SiteResults, SiteResult{
			Check:   check,
			IsIPv6:  isIPv6,
			Results: []Result{newResult},
		})
		if !status {
			go RecordEvent("site", check.Name, member.Details.Name, "", "", false, errorMsg, dataMap, isIPv6)
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
			if !status {
				go RecordEvent("site", check.Name, member.Details.Name, "", "", false, errorMsg, dataMap, isIPv6)
			}
		} else {
			if sr.Results[rIndex].Status != status {
				go RecordEvent("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap, isIPv6)
			}
			sr.Results[rIndex] = newResult
		}
	}

	publishSnapshotLocked()
}

func UpdateOfficialDomainResult(check cfg.Check, member cfg.Member, service cfg.Service, domain string,
	status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {

	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	dIndex := -1
	for i, dr := range Official.DomainResults {
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
		Official.DomainResults = append(Official.DomainResults, DomainResult{
			Check:   check,
			Service: service,
			Domain:  domain,
			IsIPv6:  isIPv6,
			Results: []Result{newResult},
		})
		if !status {
			go RecordEvent("domain", check.Name, member.Details.Name, domain, "", false, "Offline since startup", dataMap, isIPv6)
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
			if !status {
				go RecordEvent("domain", check.Name, member.Details.Name, domain, "", false, "Offline since startup", dataMap, isIPv6)
			}
		} else {
			if dr.Results[rIndex].Status != status {
				go RecordEvent("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap, isIPv6)
			}
			dr.Results[rIndex] = newResult
		}
	}

	publishSnapshotLocked()
}

func UpdateOfficialEndpointResult(check cfg.Check, member cfg.Member, service cfg.Service, domain string, endpoint string,
	status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {

	Official.Mu.Lock()
	defer Official.Mu.Unlock()

	eIndex := -1
	for i, er := range Official.EndpointResults {
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
		Official.EndpointResults = append(Official.EndpointResults, EndpointResult{
			Check:   check,
			Service: service,
			RpcUrl:  endpoint,
			Domain:  domain,
			IsIPv6:  isIPv6,
			Results: []Result{newResult},
		})
		if !status {
			go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, false, "Offline since startup", dataMap, isIPv6)
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
			if !status {
				go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, false, "Offline since startup", dataMap, isIPv6)
			}
		} else {
			if er.Results[rIndex].Status != status {
				go RecordEvent("endpoint", check.Name, member.Details.Name, domain, endpoint, status, errorMsg, dataMap, isIPv6)
			}
			er.Results[rIndex] = newResult
		}
	}

	publishSnapshotLocked()
}

func latestStatusFromResults(results []Result, memberName string) (found bool, latest bool, newest time.Time) {
	for _, r := range results {
		if r.Member.Details.Name != memberName {
			continue
		}
		if !found || r.Checktime.After(newest) {
			found = true
			latest = r.Status
			newest = r.Checktime
		}
	}
	return
}

func GetOfficialSiteStatus(checkName, memberName string, isIPv6 bool) (bool, bool) {
	sites, _, _ := GetOfficialResults()

	var newest time.Time
	var latest bool
	found := false

	for _, sr := range sites {
		if sr.Check.Name != checkName || sr.IsIPv6 != isIPv6 {
			continue
		}
		if ok, st, ct := latestStatusFromResults(sr.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

func GetOfficialDomainStatus(checkName, memberName, domain string, isIPv6 bool) (bool, bool) {
	_, domains, _ := GetOfficialResults()

	var newest time.Time
	var latest bool
	found := false

	for _, dr := range domains {
		if dr.Check.Name != checkName || dr.Domain != domain || dr.IsIPv6 != isIPv6 {
			continue
		}
		if ok, st, ct := latestStatusFromResults(dr.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

func GetOfficialEndpointStatus(checkName, memberName, domain, endpoint string, isIPv6 bool) (bool, bool) {
	_, _, endpoints := GetOfficialResults()

	var newest time.Time
	var latest bool
	found := false

	for _, er := range endpoints {
		if er.Check.Name != checkName || er.Domain != domain || er.RpcUrl != endpoint || er.IsIPv6 != isIPv6 {
			continue
		}
		if ok, st, ct := latestStatusFromResults(er.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

func publishSnapshotLocked() {
	snap := BuildSnapshot(
		Official.SiteResults,
		Official.DomainResults,
		Official.EndpointResults,
	)
	SetOfficialSnapshot(snap)
}
