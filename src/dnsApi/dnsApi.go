package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"ibp-geodns/src/dnsApi/api"
)

// Removed references to nats
var version = "0.7.0"

func main() {
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "IBP-GeoDNS DNS backend v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Load config and init MaxMind
	cfg.Init(*cfgFile)
	max.Init()

	// Start polling serviceMonitor for official results
	startMonitorPoller()

	// Launch DNS API
	api.Init()

	// Keep running
	for {
		time.Sleep(60 * time.Second)
	}
}

func startMonitorPoller() {
	go func() {
		for {
			time.Sleep(15 * time.Second)
			updateOfficialResultsSnapshot()
		}
	}()
}

func updateOfficialResultsSnapshot() {
	c := cfg.GetConfig()
	url := fmt.Sprintf("http://%s:%s/results",
		c.Local.MonitorApi.ListenAddress,
		c.Local.MonitorApi.ListenPort,
	)

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Log(log.Warn, "Failed to fetch official results from serviceMonitor: %v", err)
		return
	}
	defer resp.Body.Close()

	var tmp api.OfficialResults
	err = api.DecodeJSONBody(resp.Body, &tmp)
	if err != nil {
		log.Log(log.Warn, "Failed to decode official results: %v", err)
		return
	}

	api.UpdateOfficialResultsSnapshot(tmp)
	log.Log(log.Debug, "DNS backend updated local official results snapshot.")
}
