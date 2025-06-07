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

	// The DNS expects a specific JSON structure with MemberName as a string
	// We need to transform from our internal structure (which has full Member objects)
	// to what the DNS expects

	// Convert site results
	var apiSites []interface{}
	for _, site := range sites {
		apiSite := map[string]interface{}{
			"CheckName": site.Check.Name,
			"IsIPv6":    site.IsIPv6,
			"Results":   []interface{}{},
		}

		results := []interface{}{}
		for _, result := range site.Results {
			apiResult := map[string]interface{}{
				"MemberName": result.Member.Details.Name, // Extract name from Member object
				"Status":     result.Status,
				"ErrorText":  result.ErrorText,
				"Data":       result.Data,
				"IsIPv6":     result.IsIPv6,
			}
			results = append(results, apiResult)
		}
		apiSite["Results"] = results
		apiSites = append(apiSites, apiSite)
	}

	// Convert domain results
	var apiDomains []interface{}
	for _, domain := range domains {
		apiDomain := map[string]interface{}{
			"CheckName": domain.Check.Name,
			"Domain":    domain.Domain,
			"IsIPv6":    domain.IsIPv6,
			"Results":   []interface{}{},
		}

		results := []interface{}{}
		for _, result := range domain.Results {
			apiResult := map[string]interface{}{
				"MemberName": result.Member.Details.Name, // Extract name from Member object
				"Status":     result.Status,
				"ErrorText":  result.ErrorText,
				"Data":       result.Data,
				"IsIPv6":     result.IsIPv6,
			}
			results = append(results, apiResult)
		}
		apiDomain["Results"] = results
		apiDomains = append(apiDomains, apiDomain)
	}

	// Convert endpoint results
	var apiEndpoints []interface{}
	for _, endpoint := range endpoints {
		apiEndpoint := map[string]interface{}{
			"CheckName": endpoint.Check.Name,
			"Domain":    endpoint.Domain,
			"RpcUrl":    endpoint.RpcUrl,
			"IsIPv6":    endpoint.IsIPv6,
			"Results":   []interface{}{},
		}

		results := []interface{}{}
		for _, result := range endpoint.Results {
			apiResult := map[string]interface{}{
				"MemberName": result.Member.Details.Name, // Extract name from Member object
				"Status":     result.Status,
				"ErrorText":  result.ErrorText,
				"Data":       result.Data,
				"IsIPv6":     result.IsIPv6,
			}
			results = append(results, apiResult)
		}
		apiEndpoint["Results"] = results
		apiEndpoints = append(apiEndpoints, apiEndpoint)
	}

	// Return the properly formatted response
	out := map[string]interface{}{
		"SiteResults":     apiSites,
		"DomainResults":   apiDomains,
		"EndpointResults": apiEndpoints,
	}

	// Log some debug info
	offlineCount := 0
	for _, site := range sites {
		for _, result := range site.Results {
			if !result.Status {
				offlineCount++
				log.Log(log.Debug, "Monitor API: Returning offline member %s for site check %s",
					result.Member.Details.Name, site.Check.Name)
			}
		}
	}
	log.Log(log.Debug, "Monitor API: Returning %d site results with %d offline members", len(sites), offlineCount)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
