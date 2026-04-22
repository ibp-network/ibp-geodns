package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	dat "github.com/ibp-network/ibp-geodns-libs/data"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func Init() error {
	log.Log(log.Info, "DNS Package initializing...")

	c := cfg.GetConfig()

	syncRuntimeConfigCaches(true)
	startRuntimeConfigWatcher()

	// Setup NATS subscription for runtime override updates
	setupNatsCountryOverrideHandler()

	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", dnsApiRouter)
	dnsApi.HandleFunc("/process", handleManualUsageProcess)

	host := strings.TrimSpace(c.Local.DnsApi.ListenAddress)
	port := strings.TrimSpace(c.Local.DnsApi.ListenPort)
	if port == "" {
		return fmt.Errorf("DNS API ListenPort is empty in config")
	}
	if host == "" {
		host = "0.0.0.0"
	}
	addr := host + ":" + port
	log.Log(log.Info, "Starting DNS API server on %s", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: dnsApi,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("DNS API server failed to listen on %s: %w", addr, err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Log(log.Fatal, "DNS API server stopped on %s: %v", addr, err)
			os.Exit(1)
		}
	}()

	return nil
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
