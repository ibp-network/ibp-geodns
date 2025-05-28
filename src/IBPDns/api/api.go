package api

import (
	"encoding/json"
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// Init initializes and starts the DNS API for PowerDNS.
func Init() {
	log.Log(log.Info, "DNS Package initializing...")

	// Fetch the current configuration
	c := cfg.GetConfig()

	// Load and store static DNS records
	StaticDNSEntries()

	// Gather and store TLD records
	populateTLDRecords()

	// Rebuild dynamic service records for DNS routing
	RebuildServiceRecords()

	// Create a new HTTP multiplexer
	dnsApi := http.NewServeMux()

	// Primary DNS route for PowerDNS queries
	dnsApi.HandleFunc("/dns", dnsApiRouter)

	// Manual usage processing route (fixed default date to "today")
	dnsApi.HandleFunc("/process", handleManualUsageProcess)

	// Start listening on the configured IP/port
	log.Log(log.Info, "Starting DNS API server on %s:%s",
		c.Local.DnsApi.ListenAddress,
		c.Local.DnsApi.ListenPort,
	)

	// If the OS supports dual-stack, also try "::" if ListenAddress is 0.0.0.0
	go http.ListenAndServe(
		c.Local.DnsApi.ListenAddress+":"+c.Local.DnsApi.ListenPort,
		dnsApi,
	)
}

// handleManualUsageProcess triggers usage processing for a specified date
// or defaults to the current UTC date if none is supplied.
func handleManualUsageProcess(w http.ResponseWriter, r *http.Request) {
	dateParam := r.URL.Query().Get("date")
	if dateParam == "" {
		// Previously used "yesterday," changed to "today" so we actually match usage data from the same day.
		dateParam = time.Now().UTC().Format("2006-01-02")
	}

	// Log that a manual usage process call was received
	log.Log(log.Info, "Manual usage processing triggered for date: %s", dateParam)

	// Run usage processing for the specified date
	dat.ProcessDailyUsage(dateParam)

	// Respond with a JSON status
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status":  "success",
		"message": "Usage processing triggered for " + dateParam,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Log(log.Error, "Failed to encode JSON in handleManualUsageProcess: %v", err)
	}
}
