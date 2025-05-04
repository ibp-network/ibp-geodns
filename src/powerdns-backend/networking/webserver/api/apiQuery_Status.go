package api

import (
	"encoding/json"
	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data"
	l "ibp-geodns/src/common/logging"
	"ibp-geodns/src/common/networking/geoip"
	"net/http"
)

func ApiQuery_Status(w http.ResponseWriter, r *http.Request) ApiResponse {
	var req ApiRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		l.Log(l.Error, "Bad request: %v", http.StatusBadRequest)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return ApiResponse{Result: ""}
	}

	var res ApiResponse

	switch req.Method {
	case "byMember":
		res = apiQuery_StatusByMember()
	case "byDomain":
		res = apiQuery_StatusByDomain()
	case "byService":
		res = apiQuery_StatusByService()
	default:
		res = ApiResponse{Error: "Invalid request"}
	}

	return res
}

func apiQuery_StatusByMember() ApiResponse {
	c := config.GetConfig()

	var results = data.OfficialResults{
		SiteResults:     make([]data.SiteResult, 0),
		DomainResults:   make([]data.DomainResult, 0),
		EndpointResults: make([]data.EndpointResult, 0),
	}

	sites, domains, endpoints := data.GetOfficialResults()

	results.SiteResults = sites
	results.DomainResults = domains
	results.EndpointResults = endpoints

	byMember := make(map[string]map[string][]CheckResult)

	// Initialize each member map
	for memberName := range c.Members {
		byMember[memberName] = make(map[string][]CheckResult)
	}

	addCheck := func(memberName, domainName, checkName string, status bool, errorText string, data map[string]interface{}) {
		existing := byMember[memberName][domainName]
		for _, e := range existing {
			if e.CheckName == checkName {
				return
			}
		}
		byMember[memberName][domainName] = append(byMember[memberName][domainName], CheckResult{
			CheckName: checkName,
			Status:    status,
			ErrorText: errorText,
			Data:      data,
		})
	}

	// Add site checks for all members under "site" domain
	for i := range results.SiteResults {
		siteRes := &results.SiteResults[i]
		checkName := siteRes.Check.Name
		for j := range siteRes.Results {
			r := siteRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, "site", checkName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Domain checks
	for i := range results.DomainResults {
		domainRes := &results.DomainResults[i]
		domainName := domainRes.Domain
		checkName := domainRes.Check.Name
		for j := range domainRes.Results {
			r := domainRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, domainName, checkName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Endpoint checks
	for i := range results.EndpointResults {
		endpointRes := &results.EndpointResults[i]
		domainName := endpointRes.Domain
		checkName := endpointRes.Check.Name
		endpointID := endpointRes.RpcUrl + endpointRes.Path
		fullCheckName := checkName + "::" + endpointID
		for j := range endpointRes.Results {
			r := endpointRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, domainName, fullCheckName, r.Status, r.ErrorText, r.Data)
		}
	}

	return ApiResponse{Result: byMember}
}

func apiQuery_StatusByDomain() ApiResponse {
	c := config.GetConfig()

	var results = data.OfficialResults{
		SiteResults:     make([]data.SiteResult, 0),
		DomainResults:   make([]data.DomainResult, 0),
		EndpointResults: make([]data.EndpointResult, 0),
	}

	sites, domains, endpoints := data.GetOfficialResults()

	results.SiteResults = sites
	results.DomainResults = domains
	results.EndpointResults = endpoints

	byDomain := make(map[string]map[string][]CheckResult)

	addCheck := func(domainName, memberName, checkName string, status bool, errorText string, data map[string]interface{}) {
		dmap, ok := byDomain[domainName]
		if !ok {
			dmap = make(map[string][]CheckResult)
			byDomain[domainName] = dmap
		}
		existing := dmap[memberName]
		for _, e := range existing {
			if e.CheckName == checkName {
				return
			}
		}
		dmap[memberName] = append(dmap[memberName], CheckResult{
			CheckName: checkName,
			Status:    status,
			ErrorText: errorText,
			Data:      data,
		})
	}

	// Gather all known domains from StaticDNS and Services
	domainSet := make(map[string]struct{})
	for _, dnsRecord := range c.StaticDNS {
		domainName := dnsRecord.QName
		domainSet[domainName] = struct{}{}
	}
	for _, svc := range c.Services {
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				u := geoip.ParseUrl(rpcUrl)
				if u.Domain != "" {
					domainSet[u.Domain] = struct{}{}
				}
			}
		}
	}

	// Initialize domains with all members
	for domainName := range domainSet {
		byDomain[domainName] = make(map[string][]CheckResult)
		for memberName := range c.Members {
			byDomain[domainName][memberName] = []CheckResult{}
		}
	}

	// Add site checks everywhere first
	for domainName := range domainSet {
		for i := range results.SiteResults {
			siteRes := &results.SiteResults[i]
			checkName := siteRes.Check.Name
			for j := range siteRes.Results {
				r := siteRes.Results[j]
				memberName := r.Member.Details.Name
				addCheck(domainName, memberName, checkName, r.Status, r.ErrorText, r.Data)
			}
		}
	}

	// Domain checks
	for i := range results.DomainResults {
		domainRes := &results.DomainResults[i]
		domainName := domainRes.Domain
		checkName := domainRes.Check.Name

		// If domainName not in domainSet, add it now and also add site checks
		if _, ok := domainSet[domainName]; !ok {
			domainSet[domainName] = struct{}{}
			byDomain[domainName] = make(map[string][]CheckResult)
			for memberName := range c.Members {
				byDomain[domainName][memberName] = []CheckResult{}
			}
			// Add site checks for this newly found domain
			for si := range results.SiteResults {
				siteRes := &results.SiteResults[si]
				sCheckName := siteRes.Check.Name
				for sj := range siteRes.Results {
					sr := siteRes.Results[sj]
					mn := sr.Member.Details.Name
					addCheck(domainName, mn, sCheckName, sr.Status, sr.ErrorText, sr.Data)
				}
			}
		}

		for j := range domainRes.Results {
			r := domainRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(domainName, memberName, checkName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Endpoint checks
	for i := range results.EndpointResults {
		endpointRes := &results.EndpointResults[i]
		domainName := endpointRes.Domain
		checkName := endpointRes.Check.Name
		endpointID := endpointRes.RpcUrl + endpointRes.Path
		fullCheckName := checkName + "::" + endpointID

		// If domainName not in domainSet, add it and site checks
		if _, ok := domainSet[domainName]; !ok {
			domainSet[domainName] = struct{}{}
			byDomain[domainName] = make(map[string][]CheckResult)
			for memberName := range c.Members {
				byDomain[domainName][memberName] = []CheckResult{}
			}
			// Add site checks for this newly found domain
			for si := range results.SiteResults {
				siteRes := &results.SiteResults[si]
				sCheckName := siteRes.Check.Name
				for sj := range siteRes.Results {
					sr := siteRes.Results[sj]
					mn := sr.Member.Details.Name
					addCheck(domainName, mn, sCheckName, sr.Status, sr.ErrorText, sr.Data)
				}
			}
		}

		for j := range endpointRes.Results {
			r := endpointRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(domainName, memberName, fullCheckName, r.Status, r.ErrorText, r.Data)
		}
	}

	return ApiResponse{Result: byDomain}
}

func apiQuery_StatusByService() ApiResponse {
	c := config.GetConfig()

	var results = data.OfficialResults{
		SiteResults:     make([]data.SiteResult, 0),
		DomainResults:   make([]data.DomainResult, 0),
		EndpointResults: make([]data.EndpointResult, 0),
	}

	sites, domains, endpoints := data.GetOfficialResults()

	results.SiteResults = sites
	results.DomainResults = domains
	results.EndpointResults = endpoints

	byService := make(map[string]map[string][]CheckResult)

	addCheck := func(serviceName, memberName, checkName string, status bool, errorText string, data map[string]interface{}) {
		smap, ok := byService[serviceName]
		if !ok {
			smap = make(map[string][]CheckResult)
			byService[serviceName] = smap
		}
		existing := smap[memberName]
		for _, e := range existing {
			if e.CheckName == checkName {
				return
			}
		}
		smap[memberName] = append(smap[memberName], CheckResult{
			CheckName: checkName,
			Status:    status,
			ErrorText: errorText,
			Data:      data,
		})
	}

	// Gather all service names from config
	serviceNames := make([]string, 0, len(c.Services))
	for _, svc := range c.Services {
		serviceNames = append(serviceNames, svc.Configuration.Name)
	}

	// Initialize services and members
	for _, svcName := range serviceNames {
		byService[svcName] = make(map[string][]CheckResult)
		for memberName := range c.Members {
			byService[svcName][memberName] = []CheckResult{}
		}
	}

	// Add site checks first to all services/members
	for i := range results.SiteResults {
		siteRes := &results.SiteResults[i]
		checkName := siteRes.Check.Name
		for j := range siteRes.Results {
			r := siteRes.Results[j]
			memberName := r.Member.Details.Name
			for _, svcName := range serviceNames {
				addCheck(svcName, memberName, checkName, r.Status, r.ErrorText, r.Data)
			}
		}
	}

	// Domain checks
	for i := range results.DomainResults {
		domainRes := &results.DomainResults[i]
		serviceName := domainRes.Service.Configuration.Name
		checkName := domainRes.Check.Name
		domainName := domainRes.Domain
		fullCheckName := checkName + "::" + domainName

		// If the serviceName wasn't in our list, add it and site checks
		if _, ok := byService[serviceName]; !ok {
			byService[serviceName] = make(map[string][]CheckResult)
			for memberName := range c.Members {
				byService[serviceName][memberName] = []CheckResult{}
			}
			// Add site checks for new service
			for si := range results.SiteResults {
				siteRes := &results.SiteResults[si]
				sCheckName := siteRes.Check.Name
				for sj := range siteRes.Results {
					sr := siteRes.Results[sj]
					mn := sr.Member.Details.Name
					addCheck(serviceName, mn, sCheckName, sr.Status, sr.ErrorText, sr.Data)
				}
			}
		}

		for j := range domainRes.Results {
			r := domainRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(serviceName, memberName, fullCheckName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Endpoint checks
	for i := range results.EndpointResults {
		endpointRes := &results.EndpointResults[i]
		serviceName := endpointRes.Service.Configuration.Name
		checkName := endpointRes.Check.Name
		domainName := endpointRes.Domain
		endpointID := endpointRes.RpcUrl + endpointRes.Path
		fullCheckName := checkName + "::" + domainName + "::" + endpointID

		// If the serviceName wasn't in our list
		if _, ok := byService[serviceName]; !ok {
			byService[serviceName] = make(map[string][]CheckResult)
			for memberName := range c.Members {
				byService[serviceName][memberName] = []CheckResult{}
			}
			// Add site checks for new service
			for si := range results.SiteResults {
				siteRes := &results.SiteResults[si]
				sCheckName := siteRes.Check.Name
				for sj := range siteRes.Results {
					sr := siteRes.Results[sj]
					mn := sr.Member.Details.Name
					addCheck(serviceName, mn, sCheckName, sr.Status, sr.ErrorText, sr.Data)
				}
			}
		}

		for j := range endpointRes.Results {
			r := endpointRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(serviceName, memberName, fullCheckName, r.Status, r.ErrorText, r.Data)
		}
	}

	return ApiResponse{Result: byService}
}
