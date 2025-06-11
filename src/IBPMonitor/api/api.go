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

/*
   ---------------------------------------------------------------------------
   Helpers ‒ merge local + official results
   ---------------------------------------------------------------------------
*/

// build a unique string key for each result‑set so we can deduplicate easily.
func keySite(checkName string, isIPv6 bool) string {
	if isIPv6 {
		return checkName + "|v6"
	}
	return checkName + "|v4"
}

func keyDomain(checkName, domain string, isIPv6 bool) string {
	if isIPv6 {
		return checkName + "|" + domain + "|v6"
	}
	return checkName + "|" + domain + "|v4"
}

func keyEndpoint(checkName, domain, rpc string, isIPv6 bool) string {
	if isIPv6 {
		return checkName + "|" + domain + "|" + rpc + "|v6"
	}
	return checkName + "|" + domain + "|" + rpc + "|v4"
}

// mergeSiteResults produces the union of official + local site results.
func mergeSiteResults(off, loc []dat.SiteResult) []dat.SiteResult {
	m := make(map[string]*dat.SiteResult)

	// helper to copy a dat.SiteResult so we don't mutate the originals
	copyResult := func(src dat.SiteResult) dat.SiteResult {
		dst := src
		dst.Results = append([]dat.Result(nil), src.Results...)
		return dst
	}

	for _, s := range off {
		k := keySite(s.Check.Name, s.IsIPv6)
		c := copyResult(s)
		m[k] = &c
	}
	for _, s := range loc {
		k := keySite(s.Check.Name, s.IsIPv6)
		target, ok := m[k]
		if !ok {
			c := copyResult(s)
			m[k] = &c
			continue
		}

		// merge member‑results
		for _, r := range s.Results {
			found := false
			for i, exist := range target.Results {
				if exist.Member.Details.Name == r.Member.Details.Name {
					found = true
					// keep whichever result is NEWER by Checktime
					if r.Checktime.After(exist.Checktime) {
						target.Results[i] = r
					}
					break
				}
			}
			if !found {
				target.Results = append(target.Results, r)
			}
		}
	}

	merged := make([]dat.SiteResult, 0, len(m))
	for _, v := range m {
		merged = append(merged, *v)
	}
	return merged
}

// mergeDomainResults – similar strategy for domain results
func mergeDomainResults(off, loc []dat.DomainResult) []dat.DomainResult {
	m := make(map[string]*dat.DomainResult)

	copyResult := func(src dat.DomainResult) dat.DomainResult {
		dst := src
		dst.Results = append([]dat.Result(nil), src.Results...)
		return dst
	}

	for _, d := range off {
		k := keyDomain(d.Check.Name, d.Domain, d.IsIPv6)
		c := copyResult(d)
		m[k] = &c
	}
	for _, d := range loc {
		k := keyDomain(d.Check.Name, d.Domain, d.IsIPv6)
		target, ok := m[k]
		if !ok {
			c := copyResult(d)
			m[k] = &c
			continue
		}
		for _, r := range d.Results {
			found := false
			for i, exist := range target.Results {
				if exist.Member.Details.Name == r.Member.Details.Name {
					found = true
					if r.Checktime.After(exist.Checktime) {
						target.Results[i] = r
					}
					break
				}
			}
			if !found {
				target.Results = append(target.Results, r)
			}
		}
	}

	merged := make([]dat.DomainResult, 0, len(m))
	for _, v := range m {
		merged = append(merged, *v)
	}
	return merged
}

// mergeEndpointResults – union of endpoint results
func mergeEndpointResults(off, loc []dat.EndpointResult) []dat.EndpointResult {
	m := make(map[string]*dat.EndpointResult)

	copyResult := func(src dat.EndpointResult) dat.EndpointResult {
		dst := src
		dst.Results = append([]dat.Result(nil), src.Results...)
		return dst
	}

	for _, e := range off {
		k := keyEndpoint(e.Check.Name, e.Domain, e.RpcUrl, e.IsIPv6)
		c := copyResult(e)
		m[k] = &c
	}
	for _, e := range loc {
		k := keyEndpoint(e.Check.Name, e.Domain, e.RpcUrl, e.IsIPv6)
		target, ok := m[k]
		if !ok {
			c := copyResult(e)
			m[k] = &c
			continue
		}
		for _, r := range e.Results {
			found := false
			for i, exist := range target.Results {
				if exist.Member.Details.Name == r.Member.Details.Name {
					found = true
					if r.Checktime.After(exist.Checktime) {
						target.Results[i] = r
					}
					break
				}
			}
			if !found {
				target.Results = append(target.Results, r)
			}
		}
	}

	merged := make([]dat.EndpointResult, 0, len(m))
	for _, v := range m {
		merged = append(merged, *v)
	}
	return merged
}

/*
   ---------------------------------------------------------------------------
   Public API initialisation
   ---------------------------------------------------------------------------
*/

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

/*
   ---------------------------------------------------------------------------
   /results handler
   ---------------------------------------------------------------------------
*/

func handleResults(w http.ResponseWriter, r *http.Request) {
	// 1) fetch snapshots
	offSites, offDomains, offEndpoints := dat.GetOfficialResults()
	locSites, locDomains, locEndpoints := dat.GetLocalResults()

	// 2) merge local + official
	sites := mergeSiteResults(offSites, locSites)
	domains := mergeDomainResults(offDomains, locDomains)
	endpoints := mergeEndpointResults(offEndpoints, locEndpoints)

	/*
	   -----------------------------------------------------------------------
	   Convert to the lighter JSON structures expected by downstream callers.
	   We include Checktime so DNS nodes can decide which status is newest.
	   -----------------------------------------------------------------------
	*/

	// -------- Site ---------------------------------------------------------
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
				"Checktime":  res.Checktime.Format(time.RFC3339),
			})
		}
		out["Results"] = results
		apiSites = append(apiSites, out)
	}

	// -------- Domain -------------------------------------------------------
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
				"Checktime":  res.Checktime.Format(time.RFC3339),
			})
		}
		out["Results"] = results
		apiDomains = append(apiDomains, out)
	}

	// -------- Endpoint -----------------------------------------------------
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
				"Checktime":  res.Checktime.Format(time.RFC3339),
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
