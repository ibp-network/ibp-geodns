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

// ---------------- key helpers ------------------------------------------------
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

// ---------------- build OFFLINE site -----------------------------------------
func buildOfflineSiteResults(in []dat.SiteResult) []dat.SiteResult {
	m := make(map[string]*dat.SiteResult)
	for _, s := range in {
		k := keySite(s.Check.Name, s.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.SiteResult{Check: s.Check, IsIPv6: s.IsIPv6}
		}
		for _, r := range s.Results {
			if r.Status { // keep ONLY offline
				continue
			}
			dst := m[k]
			replace := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					replace = false
					break
				}
			}
			if replace {
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

// ---------------- build OFFLINE domain ---------------------------------------
func buildOfflineDomainResults(in []dat.DomainResult) []dat.DomainResult {
	m := make(map[string]*dat.DomainResult)
	for _, d := range in {
		k := keyDomain(d.Check.Name, d.Domain, d.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.DomainResult{Check: d.Check, Service: d.Service, Domain: d.Domain, IsIPv6: d.IsIPv6}
		}
		for _, r := range d.Results {
			if r.Status {
				continue
			}
			dst := m[k]
			replace := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					replace = false
					break
				}
			}
			if replace {
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

// ---------------- build OFFLINE endpoint -------------------------------------
func buildOfflineEndpointResults(in []dat.EndpointResult) []dat.EndpointResult {
	m := make(map[string]*dat.EndpointResult)
	for _, e := range in {
		k := keyEndpoint(e.Check.Name, e.Domain, e.RpcUrl, e.IsIPv6)
		if _, ok := m[k]; !ok {
			m[k] = &dat.EndpointResult{Check: e.Check, Service: e.Service, Domain: e.Domain, RpcUrl: e.RpcUrl, IsIPv6: e.IsIPv6}
		}
		for _, r := range e.Results {
			if r.Status {
				continue
			}
			dst := m[k]
			replace := true
			for i, ex := range dst.Results {
				if ex.Member.Details.Name == r.Member.Details.Name {
					if r.Checktime.After(ex.Checktime) {
						dst.Results[i] = r
					}
					replace = false
					break
				}
			}
			if replace {
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
   API start‑up
   ---------------------------------------------------------------------------
*/

func Init() {
	c := cfg.GetConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/results", handleResults)

	log.Log(log.Info, "Starting serviceMonitor API on %s:%s",
		c.Local.MonitorApi.ListenAddress,
		c.Local.MonitorApi.ListenPort)

	go http.ListenAndServe(
		c.Local.MonitorApi.ListenAddress+":"+c.Local.MonitorApi.ListenPort,
		mux,
	)
}

/*
---------------------------------------------------------------------------
/results – returns ONLY official **offline** records
---------------------------------------------------------------------------
*/
func handleResults(w http.ResponseWriter, r *http.Request) {
	offSites, offDomains, offEndpoints := dat.GetOfficialResults()

	siteOffline := buildOfflineSiteResults(offSites)
	domainOffline := buildOfflineDomainResults(offDomains)
	endpointOffline := buildOfflineEndpointResults(offEndpoints)

	// small helper to flatten []dat.Result
	toSlim := func(res []dat.Result) []interface{} {
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

	// Site
	apiSites := make([]interface{}, 0, len(siteOffline))
	for _, s := range siteOffline {
		apiSites = append(apiSites, map[string]interface{}{
			"CheckName": s.Check.Name,
			"IsIPv6":    s.IsIPv6,
			"Results":   toSlim(s.Results),
		})
	}

	// Domain
	apiDomains := make([]interface{}, 0, len(domainOffline))
	for _, d := range domainOffline {
		apiDomains = append(apiDomains, map[string]interface{}{
			"CheckName": d.Check.Name,
			"Domain":    d.Domain,
			"IsIPv6":    d.IsIPv6,
			"Results":   toSlim(d.Results),
		})
	}

	// Endpoint
	apiEndpoints := make([]interface{}, 0, len(endpointOffline))
	for _, e := range endpointOffline {
		apiEndpoints = append(apiEndpoints, map[string]interface{}{
			"CheckName": e.Check.Name,
			"Domain":    e.Domain,
			"RpcUrl":    e.RpcUrl,
			"IsIPv6":    e.IsIPv6,
			"Results":   toSlim(e.Results),
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
