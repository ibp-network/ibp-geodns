package main

import (
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	api "ibp-geodns/src/mgmtApi/api"
)

// main starts your standalone mgmt-api server.
func main() {
	// Initialize config (adjust path if needed)
	cfgPath := "D:\\Sync\\Projects\\stake.plus\\code\\ibp-geodns-v2\\config\\config.json"
	cfg.Init(cfgPath)

	// Optionally set log level
	log.SetLogLevel(log.Info)

	// Initialize mgmt api
	api.Init()

	// Build routes
	mux := http.NewServeMux()

	// Old endpoints, now served by mgmt-api
	mux.HandleFunc("/api/billing", api.HandleApiQuery)
	mux.HandleFunc("/api/member", api.HandleApiQuery)
	mux.HandleFunc("/api/status", api.HandleApiQuery)
	mux.HandleFunc("/api/usage", api.HandleApiQuery)

	// Grab host/port from config
	addr := cfg.GetConfig().Local.MgmtApi.ListenAddress + ":" + cfg.GetConfig().Local.MgmtApi.ListenPort

	// Run server
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Log(log.Fatal, "Error starting mgmt-api: %v", err)
	}

	go loop()
}

func loop() {
	for {
		time.Sleep(60 * time.Second)
	}
}
