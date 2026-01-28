package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	dat "github.com/ibp-network/ibp-geodns-libs/data"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func Init() {
	log.Log(log.Info, "DNS Package initializing...")

	c := cfg.GetConfig()

	StaticDNSEntries()
	populateTLDRecords()
	RebuildServiceRecords()
	
	// Load country code overrides from config if available
	LoadCountryOverridesFromConfigFile()

	// Setup NATS subscription for runtime override updates
	setupNatsCountryOverrideHandler()

	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", dnsApiRouter)
	dnsApi.HandleFunc("/process", handleManualUsageProcess)

	host := strings.TrimSpace(c.Local.DnsApi.ListenAddress)
	port := strings.TrimSpace(c.Local.DnsApi.ListenPort)
	if port == "" {
		log.Log(log.Fatal, "DNS API ListenPort is empty in config; cannot start server")
		return
	}
	if host == "" {
		host = "0.0.0.0"
	}
	addr := host + ":" + port
	log.Log(log.Info, "Starting DNS API server on %s", addr)

	// Block here and fail fast if bind/listen fails; systemd will restart the service.
	if err := http.ListenAndServe(addr, dnsApi); err != nil {
		log.Log(log.Fatal, "DNS API server failed to start on %s: %v", addr, err)
	}
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
