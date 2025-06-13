package api

import (
	"encoding/json"
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

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

func newestOnly(results []dat.Result) map[string]dat.Result {
	out := make(map[string]dat.Result)
	for _, r := range results {
		name := r.Member.Details.Name
		if prev, ok := out[name]; !ok || r.Checktime.After(prev.Checktime) {
			out[name] = r
		}
	}
	return out
}

func sliceFromMap(m map[string]dat.Result, onlyOffline bool) []dat.Result {
	out := make([]dat.Result, 0, len(m))
	for _, r := range m {
		if onlyOffline && r.Status {
			continue
		}
		out = append(out, r)
	}
	return out
}

func buildOfflineSiteResults(input []dat.SiteResult) []dat.SiteResult {
	res := make(map[string]*dat.SiteResult)

	for _, src := range input {
		k := keySite(src.Check.Name, src.IsIPv6)
		if _, ok := res[k]; !ok {
			res[k] = &dat.SiteResult{Check: src.Check, IsIPv6: src.IsIPv6}
		}
		latest := newestOnly(src.Results)
		res[k].Results = append(res[k].Results, sliceFromMap(latest, true)...)
	}

	out := make([]dat.SiteResult, 0, len(res))
	for _, v := range res {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

func buildOfflineDomainResults(input []dat.DomainResult) []dat.DomainResult {
	res := make(map[string]*dat.DomainResult)

	for _, src := range input {
		k := keyDomain(src.Check.Name, src.Domain, src.IsIPv6)
		if _, ok := res[k]; !ok {
			res[k] = &dat.DomainResult{Check: src.Check, Service: src.Service, Domain: src.Domain, IsIPv6: src.IsIPv6}
		}
		latest := newestOnly(src.Results)
		res[k].Results = append(res[k].Results, sliceFromMap(latest, true)...)
	}

	out := make([]dat.DomainResult, 0, len(res))
	for _, v := range res {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

func buildOfflineEndpointResults(input []dat.EndpointResult) []dat.EndpointResult {
	res := make(map[string]*dat.EndpointResult)

	for _, src := range input {
		k := keyEndpoint(src.Check.Name, src.Domain, src.RpcUrl, src.IsIPv6)
		if _, ok := res[k]; !ok {
			res[k] = &dat.EndpointResult{Check: src.Check, Service: src.Service, Domain: src.Domain, RpcUrl: src.RpcUrl, IsIPv6: src.IsIPv6}
		}
		latest := newestOnly(src.Results)
		res[k].Results = append(res[k].Results, sliceFromMap(latest, true)...)
	}

	out := make([]dat.EndpointResult, 0, len(res))
	for _, v := range res {
		if len(v.Results) > 0 {
			out = append(out, *v)
		}
	}
	return out
}

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

func handleResults(w http.ResponseWriter, r *http.Request) {
	offSites, offDomains, offEndpoints := dat.GetOfficialResults()

	apiSites := make([]interface{}, 0)
	for _, s := range buildOfflineSiteResults(offSites) {
		apiSites = append(apiSites, map[string]interface{}{
			"CheckName": s.Check.Name,
			"IsIPv6":    s.IsIPv6,
			"Results":   slimResults(s.Results),
		})
	}

	apiDomains := make([]interface{}, 0)
	for _, d := range buildOfflineDomainResults(offDomains) {
		apiDomains = append(apiDomains, map[string]interface{}{
			"CheckName": d.Check.Name,
			"Domain":    d.Domain,
			"IsIPv6":    d.IsIPv6,
			"Results":   slimResults(d.Results),
		})
	}

	apiEndpoints := make([]interface{}, 0)
	for _, e := range buildOfflineEndpointResults(offEndpoints) {
		apiEndpoints = append(apiEndpoints, map[string]interface{}{
			"CheckName": e.Check.Name,
			"Domain":    e.Domain,
			"RpcUrl":    e.RpcUrl,
			"IsIPv6":    e.IsIPv6,
			"Results":   slimResults(e.Results),
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

func slimResults(res []dat.Result) []interface{} {
	out := make([]interface{}, 0, len(res))
	for _, r := range res {
		out = append(out, map[string]interface{}{
			"MemberName": r.Member.Details.Name,
			"ErrorText":  r.ErrorText,
			"Data":       r.Data,
			"IsIPv6":     r.IsIPv6,
			"Checktime":  r.Checktime.Format(time.RFC3339),
		})
	}
	return out
}
