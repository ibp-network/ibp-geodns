// In src/IBPMonitor/api/api.go

package api

import (
	"encoding/json"
	"net/http"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// Init starts the internal API to serve or reset official results
func Init() {
	c := cfg.GetConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/results", handleResults)

	log.Log(log.Info, "Starting serviceMonitor API on %s:%s",
		c.Local.MonitorApi.ListenAddress,
		c.Local.MonitorApi.ListenPort,
	)

	go http.ListenAndServe(
		c.Local.MonitorApi.ListenAddress+":"+c.Local.MonitorApi.ListenPort,
		mux,
	)
}

func handleResults(w http.ResponseWriter, r *http.Request) {
	// Query the official results from data
	sites, domains, endpoints := dat.GetOfficialResults()

	// Convert to the format expected by IBPDns
	// The issue is that dat.Result contains full Member objects, but IBPDns expects just names

	type GenericResult struct {
		MemberName string                 `json:"MemberName"`
		Status     bool                   `json:"Status"`
		ErrorText  string                 `json:"ErrorText"`
		Data       map[string]interface{} `json:"Data"`
		IsIPv6     bool                   `json:"IsIPv6"`
	}

	type SiteResult struct {
		CheckName string          `json:"CheckName"`
		IsIPv6    bool            `json:"IsIPv6"`
		Results   []GenericResult `json:"Results"`
	}

	type DomainResult struct {
		CheckName string          `json:"CheckName"`
		Domain    string          `json:"Domain"`
		IsIPv6    bool            `json:"IsIPv6"`
		Results   []GenericResult `json:"Results"`
	}

	type EndpointResult struct {
		CheckName string          `json:"CheckName"`
		Domain    string          `json:"Domain"`
		RpcUrl    string          `json:"RpcUrl"`
		IsIPv6    bool            `json:"IsIPv6"`
		Results   []GenericResult `json:"Results"`
	}

	// Convert site results
	var apiSites []SiteResult
	for _, s := range sites {
		apiSite := SiteResult{
			CheckName: s.Check.Name,
			IsIPv6:    s.IsIPv6,
			Results:   make([]GenericResult, 0, len(s.Results)),
		}
		for _, r := range s.Results {
			apiSite.Results = append(apiSite.Results, GenericResult{
				MemberName: r.Member.Details.Name, // Extract name from Member object
				Status:     r.Status,
				ErrorText:  r.ErrorText,
				Data:       r.Data,
				IsIPv6:     r.IsIPv6,
			})
		}
		apiSites = append(apiSites, apiSite)
	}

	// Convert domain results
	var apiDomains []DomainResult
	for _, d := range domains {
		apiDomain := DomainResult{
			CheckName: d.Check.Name,
			Domain:    d.Domain,
			IsIPv6:    d.IsIPv6,
			Results:   make([]GenericResult, 0, len(d.Results)),
		}
		for _, r := range d.Results {
			apiDomain.Results = append(apiDomain.Results, GenericResult{
				MemberName: r.Member.Details.Name, // Extract name from Member object
				Status:     r.Status,
				ErrorText:  r.ErrorText,
				Data:       r.Data,
				IsIPv6:     r.IsIPv6,
			})
		}
		apiDomains = append(apiDomains, apiDomain)
	}

	// Convert endpoint results
	var apiEndpoints []EndpointResult
	for _, e := range endpoints {
		apiEndpoint := EndpointResult{
			CheckName: e.Check.Name,
			Domain:    e.Domain,
			RpcUrl:    e.RpcUrl,
			IsIPv6:    e.IsIPv6,
			Results:   make([]GenericResult, 0, len(e.Results)),
		}
		for _, r := range e.Results {
			apiEndpoint.Results = append(apiEndpoint.Results, GenericResult{
				MemberName: r.Member.Details.Name, // Extract name from Member object
				Status:     r.Status,
				ErrorText:  r.ErrorText,
				Data:       r.Data,
				IsIPv6:     r.IsIPv6,
			})
		}
		apiEndpoints = append(apiEndpoints, apiEndpoint)
	}

	// Return the properly formatted response
	out := struct {
		SiteResults     interface{} `json:"SiteResults"`
		DomainResults   interface{} `json:"DomainResults"`
		EndpointResults interface{} `json:"EndpointResults"`
	}{
		SiteResults:     apiSites,
		DomainResults:   apiDomains,
		EndpointResults: apiEndpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
