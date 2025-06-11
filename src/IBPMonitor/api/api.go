// In src/IBPMonitor/api/api.go

package api

import (
	"encoding/json"
	"net/http"
	"time"

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

// ------------------------- results handler -------------------------

func handleResults(w http.ResponseWriter, r *http.Request) {
	sites, domains, endpoints := dat.GetOfficialResults()

	/* ----------------------------------------------------------------
	   Transform internal → public JSON.
	   Now includes Checktime (RFC 3339) so downstream can decide which
	   status is the latest instead of flagging a member offline forever.
	-----------------------------------------------------------------*/

	// -------- Site --------------------------------------------------
	apiSites := make([]interface{}, 0, len(sites))
	for _, site := range sites {
		out := map[string]interface{}{
			"CheckName": site.Check.Name,
			"IsIPv6":    site.IsIPv6,
			"Results":   []interface{}{},
		}
		results := make([]interface{}, 0, len(site.Results))
		for _, res := range site.Results {
			results = append(results, map[string]interface{}{
				"MemberName": res.Member.Details.Name,
				"Status":     res.Status,
				"ErrorText":  res.ErrorText,
				"Data":       res.Data,
				"IsIPv6":     res.IsIPv6,
				"Checktime":  res.Checktime.Format(time.RFC3339), // ← NEW
			})
		}
		out["Results"] = results
		apiSites = append(apiSites, out)
	}

	// -------- Domain ------------------------------------------------
	apiDomains := make([]interface{}, 0, len(domains))
	for _, dom := range domains {
		out := map[string]interface{}{
			"CheckName": dom.Check.Name,
			"Domain":    dom.Domain,
			"IsIPv6":    dom.IsIPv6,
			"Results":   []interface{}{},
		}
		results := make([]interface{}, 0, len(dom.Results))
		for _, res := range dom.Results {
			results = append(results, map[string]interface{}{
				"MemberName": res.Member.Details.Name,
				"Status":     res.Status,
				"ErrorText":  res.ErrorText,
				"Data":       res.Data,
				"IsIPv6":     res.IsIPv6,
				"Checktime":  res.Checktime.Format(time.RFC3339), // ← NEW
			})
		}
		out["Results"] = results
		apiDomains = append(apiDomains, out)
	}

	// -------- Endpoint ---------------------------------------------
	apiEndpoints := make([]interface{}, 0, len(endpoints))
	for _, ep := range endpoints {
		out := map[string]interface{}{
			"CheckName": ep.Check.Name,
			"Domain":    ep.Domain,
			"RpcUrl":    ep.RpcUrl,
			"IsIPv6":    ep.IsIPv6,
			"Results":   []interface{}{},
		}
		results := make([]interface{}, 0, len(ep.Results))
		for _, res := range ep.Results {
			results = append(results, map[string]interface{}{
				"MemberName": res.Member.Details.Name,
				"Status":     res.Status,
				"ErrorText":  res.ErrorText,
				"Data":       res.Data,
				"IsIPv6":     res.IsIPv6,
				"Checktime":  res.Checktime.Format(time.RFC3339), // ← NEW
			})
		}
		out["Results"] = results
		apiEndpoints = append(apiEndpoints, out)
	}

	resp := map[string]interface{}{
		"SiteResults":     apiSites,
		"DomainResults":   apiDomains,
		"EndpointResults": apiEndpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
