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

var version = "0.8.0"

func main() {
	// 1) Read command-line flags for config.
	cfgFile := flag.String("config", "config.json", "Path to configuration file")
	flag.Parse()

	// 2) Check file existence
	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// 3) Initialize the config
	cfg.Init(*cfgFile)
	c := cfg.GetConfig()
	// 3a) Set log level from config
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// 4) Initialize the data layer so that:
	//    - The Stats map is created
	//    - The auto-update ticker for caches starts
	//    - MySQL usage is connected
	dat.Init()

	// 5) Initialize MaxMind
	max.Init()

	// 6) Start the DNS API (which also loads static records, TLD map, etc.)
	api.Init()

	// 7) Start poller to fetch official results from the serviceMonitor
	intervalSec := c.Local.DnsApi.RefreshIntervalSeconds
	log.Log(log.Info, "DNSAPI v%s starting... Monitoring poll interval = %d seconds", version, intervalSec)
	startServiceMonitorPoller(intervalSec)

	// 8) Keep running
	for {
		time.Sleep(60 * time.Second)
	}
}

// startServiceMonitorPoller fetches official results from serviceMonitor
func startServiceMonitorPoller(intervalSec int) {
	// One immediate fetch
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

	api.SetLocalSnapshot(tmp)
	log.Log(log.Debug, "dnsApi poller: updated local results snapshot.")
}
