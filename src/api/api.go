package api

import (
	"encoding/json"
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

func Init() {
	log.Log(log.Info, "DNS Package initializing...")

	c := cfg.GetConfig()

	StaticDNSEntries()
	populateTLDRecords()
	RebuildServiceRecords()

	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", dnsApiRouter)
	dnsApi.HandleFunc("/process", handleManualUsageProcess)

	log.Log(log.Info, "Starting DNS API server on %s:%s",
		c.Local.DnsApi.ListenAddress,
		c.Local.DnsApi.ListenPort,
	)

	go http.ListenAndServe(
		c.Local.DnsApi.ListenAddress+":"+c.Local.DnsApi.ListenPort,
		dnsApi,
	)
}

func handleManualUsageProcess(w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	if dateParam == "" {
		dateParam = time.Now().UTC().Format("2006-01-02")
	}

	log.Log(log.Info, "Manual usage processing triggered for date: %s", dateParam)

	dat.FlushUsageToDatabase(dateParam)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "success",
		"message": "Usage processing triggered for " + dateParam,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Log(log.Error, "Failed to encode JSON in handleManualUsageProcess: %v", err)
	}
}
