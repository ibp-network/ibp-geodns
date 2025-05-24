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

	// We'll define a small struct to return these in JSON
	out := struct {
		SiteResults     interface{} `json:"SiteResults"`
		DomainResults   interface{} `json:"DomainResults"`
		EndpointResults interface{} `json:"EndpointResults"`
	}{
		SiteResults:     sites,
		DomainResults:   domains,
		EndpointResults: endpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
