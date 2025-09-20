package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	dat "github.com/ibp-network/ibp-geodns-libs/data"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
	max "github.com/ibp-network/ibp-geodns-libs/maxmind"
	natsCommon "github.com/ibp-network/ibp-geodns-libs/nats"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"

	api "ibp-geodns/src/api"
)

var version = cfg.GetVersion()

func main() {
	log.Log(log.Info, "IBPDns %s starting...", version)

	cfgFile := flag.String("config", "ibpdns.json", "Path to configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	cfg.Init(*cfgFile)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "DNS API is running with log level: %s", c.Local.System.LogLevel)

	dat.Init(dat.InitOptions{UseLocalOfficialCaches: false, UseUsageStats: true})
	max.Init()

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	go startServiceMonitorPoller(c.Local.DnsApi.RefreshIntervalSeconds)

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

	api.Init()

	for {
		time.Sleep(60 * time.Second)
	}
}

func startServiceMonitorPoller(intervalSec int) {
	updateDNSMonitorSnapshot()

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	for range ticker.C {
		updateDNSMonitorSnapshot()
	}
}

func updateDNSMonitorSnapshot() {
	c := cfg.GetConfig()
	url := fmt.Sprintf("http://%s:%s/results", c.Local.DnsApi.MonitorAddress, c.Local.DnsApi.MonitorPort)

	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(url)
	if err != nil {
		log.Log(log.Error, "[Monitor Poller] GET %s error: %v", url, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Log(log.Error, "[Monitor Poller] GET %s => HTTP %d", url, resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log(log.Error, "[Monitor Poller] read body error: %v", err)
		return
	}

	var snap api.OfficialResults
	if err := json.Unmarshal(body, &snap); err != nil {
		log.Log(log.Error, "[Monitor Poller] decode JSON: %v", err)
		return
	}
	api.SetOfficialSnapshot(snap)
}
