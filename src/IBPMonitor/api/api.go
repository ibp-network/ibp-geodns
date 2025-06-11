package api

import (
	"encoding/json"
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

/*
   ---------------------------------------------------------------------------
   Build OFFICIAL‑only *offline* snapshots
   ---------------------------------------------------------------------------
*/

// ---------- key helpers ------------------------------------------------------
func keySite(chk string, v6 bool) string {
	if v6 {
		return chk + "|v6"
	}
	return chk + "|v4"
}
func keyDomain(chk, dom string, v6 bool) string {
	if v6 {
		return chk + "|" + dom + "|v6"
	}
	return chk + "|" + dom + "|v4"
}
func keyEndpoint(chk, dom, rpc string, v6 bool) string {
	if v6 {
		return chk + "|" + dom + "|" + rpc + "|v6"
	}
	return chk + "|" + dom + "|" + rpc + "|v4"
}

// ---------- OFFLINE site -----------------------------------------------------
func buildOfflineSiteResults(input []dat.SiteResult) []dat.SiteResult {
	m := make(map[string]*dat.SiteResult)
	for _, src := range input {
		k := keySite(src.Check.Name, src.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.SiteResult{Check: src.Check, IsIPv6: src.IsIPv6}
		}
		for _, r := range src.Results {
			if r.Status { // we keep *only* offline
				continue
			}
			dst := m[k]
			repl := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					repl = false
					break
				}
			}
			if repl {
				dst.Results = append(dst.Results, r)
			}
		}
	}
	out := make([]dat.SiteResult, 0, len(m))
	for _, v := range m {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

// ---------- OFFLINE domain ---------------------------------------------------
func buildOfflineDomainResults(input []dat.DomainResult) []dat.DomainResult {
	m := make(map[string]*dat.DomainResult)
	for _, src := range input {
		k := keyDomain(src.Check.Name, src.Domain, src.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.DomainResult{Check: src.Check, Service: src.Service, Domain: src.Domain, IsIPv6: src.IsIPv6}
		}
		for _, r := range src.Results {
			if r.Status {
				continue
			}
			dst := m[k]
			repl := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					repl = false
					break
				}
			}
			if repl {
				dst.Results = append(dst.Results, r)
			}
		}
	}
	out := make([]dat.DomainResult, 0, len(m))
	for _, v := range m {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

// ---------- OFFLINE endpoint -------------------------------------------------
func buildOfflineEndpointResults(input []dat.EndpointResult) []dat.EndpointResult {
	m := make(map[string]*dat.EndpointResult)
	for _, src := range input {
		k := keyEndpoint(src.Check.Name, src.Domain, src.RpcUrl, src.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.EndpointResult{Check: src.Check, Service: src.Service, Domain: src.Domain, RpcUrl: src.RpcUrl, IsIPv6: src.IsIPv6}
		}
		for _, r := range src.Results {
			if r.Status {
				continue
			}
			dst := m[k]
			repl := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					repl = false
					break
				}
			}
			if repl {
				dst.Results = append(dst.Results, r)
			}
		}
	}
	out := make([]dat.EndpointResult, 0, len(m))
	for _, v := range m {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

/*
   ---------------------------------------------------------------------------
   API initialisation
   ---------------------------------------------------------------------------
*/

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

/*
   ---------------------------------------------------------------------------
   /results – returns **official OFFLINE** results only
   ---------------------------------------------------------------------------
*/

func handleResults(w http.ResponseWriter, r *http.Request) {
	offSites, offDomains, offEndpoints := dat.GetOfficialResults()

	sites := buildOfflineSiteResults(offSites)
	domains := buildOfflineDomainResults(offDomains)
	endpoints := buildOfflineEndpointResults(offEndpoints)

	slim := func(res []dat.Result) []interface{} {
		out := make([]interface{}, 0, len(res))
		for _, r := range res {
			out = append(out, map[string]interface{}{
				"MemberName": r.Member.Details.Name,
				"Status":     r.Status,
				"ErrorText":  r.ErrorText,
				"Data":       r.Data,
				"IsIPv6":     r.IsIPv6,
				"Checktime":  r.Checktime.Format(time.RFC3339),
			})
		}
		return out
	}

	// convert
	apiSites := make([]interface{}, 0, len(sites))
	for _, s := range sites {
		apiSites = append(apiSites, map[string]interface{}{
			"CheckName": s.Check.Name,
			"IsIPv6":    s.IsIPv6,
			"Results":   slim(s.Results),
		})
	}

	apiDomains := make([]interface{}, 0, len(domains))
	for _, d := range domains {
		apiDomains = append(apiDomains, map[string]interface{}{
			"CheckName": d.Check.Name,
			"Domain":    d.Domain,
			"IsIPv6":    d.IsIPv6,
			"Results":   slim(d.Results),
		})
	}

	apiEndpoints := make([]interface{}, 0, len(endpoints))
	for _, e := range endpoints {
		apiEndpoints = append(apiEndpoints, map[string]interface{}{
			"CheckName": e.Check.Name,
			"Domain":    e.Domain,
			"RpcUrl":    e.RpcUrl,
			"IsIPv6":    e.IsIPv6,
			"Results":   slim(e.Results),
		})
	}

	resp := map[string]interface{}{
		"SiteResults":     apiSites,
		"DomainResults":   apiDomains,
		"EndpointResults": apiEndpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
