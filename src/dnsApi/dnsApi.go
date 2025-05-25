package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"ibp-geodns/src/dnsApi/api"
)

var version = "0.7.0"

func main() {
	log.Log(log.Info, "IBP-GeoDNS DNS backend v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// 1) Load config
	cfg.Init(*cfgFile)

	// 2) Now parse the configured log level from the newly loaded config
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "DNS API is running with log level: %s", c.Local.System.LogLevel)

	// 3) Initialize data usage stats but NOT local/official caches
	dat.Init(dat.InitOptions{
		UseLocalOfficialCaches: false,
		UseUsageStats:          true,
	})

	// 4) Initialize MaxMind
	max.Init()

	// 5) Launch the DNS API
	api.Init()

	// Adjust for dual stack if we see 0.0.0.0
	addr := c.Local.DnsApi.ListenAddress
	if addr == "0.0.0.0" {
		addr = "[::]"
	}
	// 6) Start listening (potentially dual-stack if OS supports it)
	log.Log(log.Info, "Starting DNS API server on %s:%s", addr, c.Local.DnsApi.ListenPort)
	dnsApi := http.DefaultServeMux // replaced in api.Init() with routes

	go http.ListenAndServe(
		fmt.Sprintf("%s:%s", addr, c.Local.DnsApi.ListenPort),
		dnsApi,
	)

	// 7) Start polling serviceMonitor for official results, if desired
	intervalSec := c.Local.DnsApi.RefreshIntervalSeconds
	log.Log(log.Info, "Starting serviceMonitor poller every %d seconds", intervalSec)
	startServiceMonitorPoller(intervalSec)

	// 8) Keep running forever
	for {
		time.Sleep(60 * time.Second)
	}
}

// startServiceMonitorPoller runs a ticker that fetches the official results
func startServiceMonitorPoller(intervalSec int) {
	updateDNSMonitorSnapshot()

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	go func() {
		for {
			<-ticker.C
			updateDNSMonitorSnapshot()
		}
	}()
}

// updateDNSMonitorSnapshot fetches official results from the monitor’s /results endpoint
func updateDNSMonitorSnapshot() {
	c := cfg.GetConfig()
	url := fmt.Sprintf("http://%s:%s/results",
		c.Local.MonitorApi.ListenAddress,
		c.Local.MonitorApi.ListenPort,
	)

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Log(log.Warn, "dnsApi poller: cannot fetch results from %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	var tmp api.OfficialResults
	decErr := api.DecodeJSONBody(resp.Body, &tmp)
	if decErr != nil {
		log.Log(log.Warn, "dnsApi poller: decode error: %v", decErr)
		return
	}

	// Store into official snapshot
	api.SetOfficialSnapshot(tmp)
	log.Log(log.Debug, "dnsApi poller: updated official results snapshot.")
}
