package mgmtApi

import (
	"net/http"

	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data"
)

// ApiQuery_Status handles /api/status requests
func ApiQuery_Status(w http.ResponseWriter, r *http.Request, req ApiRequest) ApiResponse {
	switch req.Method {
	case "byMember":
		return apiQuery_StatusByMember()
	case "byDomain":
		return apiQuery_StatusByDomain()
	case "byService":
		return apiQuery_StatusByService()
	default:
		return ApiResponse{Error: "Invalid status request"}
	}
}

// Example byMember approach: returns a structure grouped by each member
func apiQuery_StatusByMember() ApiResponse {
	c := config.GetConfig()
	sites, domains, endpoints := data.GetOfficialResults()

	byMember := make(map[string]map[string][]CheckResult)
	// init for each known member
	for memberName := range c.Members {
		byMember[memberName] = make(map[string][]CheckResult)
	}

	// Helper closure to attach a check result
	addCheck := func(memberName, domainName, checkName string, status bool, errText string, dataMap map[string]interface{}) {
		if _, ok := byMember[memberName][domainName]; !ok {
			byMember[memberName][domainName] = []CheckResult{}
		}
		byMember[memberName][domainName] = append(
			byMember[memberName][domainName],
			CheckResult{CheckName: checkName, Status: status, ErrorText: errText, Data: dataMap},
		)
	}

	// Gather site results
	for i := range sites {
		siteRes := &sites[i]
		checkName := siteRes.Check.Name
		for j := range siteRes.Results {
			r := siteRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, "site", checkName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Gather domain results
	for i := range domains {
		domRes := &domains[i]
		checkName := domRes.Check.Name
		domainName := domRes.Domain
		for j := range domRes.Results {
			r := domRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, domainName, checkName, r.Status, r.ErrorText, r.Data)
		}
	}

	// Gather endpoint results
	for i := range endpoints {
		epRes := &endpoints[i]
		checkName := epRes.Check.Name
		domainName := epRes.Domain
		endpointID := epRes.RpcUrl + epRes.Path
		combinedName := checkName + "::" + endpointID
		for j := range epRes.Results {
			r := epRes.Results[j]
			memberName := r.Member.Details.Name
			addCheck(memberName, domainName, combinedName, r.Status, r.ErrorText, r.Data)
		}
	}

	return ApiResponse{Result: byMember}
}

// Similarly, define domain or service groupings if you want:
func apiQuery_StatusByDomain() ApiResponse {
	// c := config.GetConfig() ...
	// gather OfficialResults, group them by domain
	return ApiResponse{Result: "TODO: status byDomain not yet implemented"}
}

func apiQuery_StatusByService() ApiResponse {
	// c := config.GetConfig() ...
	// gather OfficialResults, group them by service
	return ApiResponse{Result: "TODO: status byService not yet implemented"}
}
