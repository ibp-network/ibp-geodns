package api

import (
	"encoding/json"
	"net/http"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"ibp-geodns/src/serviceMonitor/monitor"
)

// Init starts an HTTP server for retrieving or resetting official results
func Init() {
	c := cfg.GetConfig()

	mux := http.NewServeMux()
	mux.HandleFunc("/results", handleResults)
	mux.HandleFunc("/reset", handleReset)

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
	results := monitor.GetOfficialResults()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	monitor.ResetOfficialResults()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"reset": true}`))
}
