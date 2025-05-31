package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	api "ibp-geodns/src/IBPDns/api"
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	natsCommon "ibp-geodns/src/common/nats"
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

	cfg.Init(*cfgFile)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "DNS API is running with log level: %s", c.Local.System.LogLevel)

	// Init data with usage stats
	dat.Init(dat.InitOptions{
		UseLocalOfficialCaches: false,
		UseUsageStats:          true,
	})

	max.Init()

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	natsCommon.State.NodeID = c.Local.Nats.NodeID
	natsCommon.State.ThisNode = natsCommon.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPDns",
	}

	if err := natsCommon.EnableDnsRole(); err != nil {
		log.Log(log.Fatal, "Failed to enable DNSApi role: %v", err)
		os.Exit(1)
	}

	// Launch DNS API
	api.Init()

	// Start polling monitor
	intervalSec := c.Local.DnsApi.RefreshIntervalSeconds
	log.Log(log.Info, "Starting serviceMonitor poller every %d seconds", intervalSec)
	startServiceMonitorPoller(intervalSec)

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

	api.SetOfficialSnapshot(tmp)
	log.Log(log.Debug, "dnsApi poller: updated official results snapshot.")
}
