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

	// Launch DNS API
	api.Init()

	// Read from config: how often to poll serviceMonitor
	c := cfg.GetConfig()
	intervalSec := c.Local.DnsApi.RefreshIntervalSeconds
	log.Log(log.Info, "Starting serviceMonitor poller every %d seconds", intervalSec)
	startServiceMonitorPoller(intervalSec)

	// Keep running
	for {
		time.Sleep(60 * time.Second)
	}
}

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

	// Store into local snapshot
	api.SetLocalSnapshot(tmp)
	log.Log(log.Debug, "dnsApi poller: updated local results snapshot.")
}
