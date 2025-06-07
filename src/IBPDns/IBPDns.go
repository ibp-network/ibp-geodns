package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
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

var version = "0.3.3"

func main() {
	log.Log(log.Info, "IBP-GeoDNS DNS backend v%s starting...", version)

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

	// Start polling monitor
	intervalSec := c.Local.DnsApi.RefreshIntervalSeconds
	log.Log(log.Info, "Starting serviceMonitor poller every %d seconds", intervalSec)
	go startServiceMonitorPoller(intervalSec)

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

// Add this to updateDNSMonitorSnapshot() in IBPDns.go to debug what we're receiving
// Also add "io" to the imports at the top of the file

func updateDNSMonitorSnapshot() {
	c := cfg.GetConfig()

	url := fmt.Sprintf("http://%s:%s/results",
		c.Local.DnsApi.MonitorAddress,
		c.Local.DnsApi.MonitorPort,
	)

	log.Log(log.Info, "[Monitor Poller] Fetching results from: %s", url)

	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Log(log.Error, "[Monitor Poller] Failed to fetch results from %s: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Log(log.Error, "[Monitor Poller] Non-OK status from monitor: %d", resp.StatusCode)
		return
	}

	// First, let's see the raw JSON response
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log(log.Error, "[Monitor Poller] Failed to read response body: %v", err)
		return
	}

	// Log a sample of the raw response (first 500 chars)
	sample := string(bodyBytes)
	if len(sample) > 500 {
		sample = sample[:500] + "..."
	}
	log.Log(log.Debug, "[Monitor Poller] Raw response sample: %s", sample)

	// Now try to decode it
	var tmp api.OfficialResults
	decErr := json.Unmarshal(bodyBytes, &tmp)
	if decErr != nil {
		log.Log(log.Error, "[Monitor Poller] Failed to decode response: %v", decErr)
		return
	}

	// Debug: Check if MemberName is actually populated
	if len(tmp.SiteResults) > 0 && len(tmp.SiteResults[0].Results) > 0 {
		firstResult := tmp.SiteResults[0].Results[0]
		log.Log(log.Debug, "[Monitor Poller] First site result - MemberName: '%s', Status: %v",
			firstResult.MemberName, firstResult.Status)
	}

	// Update the local snapshot
	api.SetOfficialSnapshot(tmp)

	// Log detailed stats
	offlineMembers := make(map[string]bool)

	// Check site results
	for _, sr := range tmp.SiteResults {
		for _, r := range sr.Results {
			if !r.Status {
				if r.MemberName == "" {
					log.Log(log.Warn, "[Monitor Poller] Found offline member with EMPTY name in site check %s", sr.CheckName)
				} else {
					offlineMembers[r.MemberName] = true
					log.Log(log.Info, "[Monitor Poller] Member %s is OFFLINE (site check %s, IPv6=%v): %s",
						r.MemberName, sr.CheckName, sr.IsIPv6, r.ErrorText)
				}
			}
		}
	}

	// Similar for domains and endpoints...

	log.Log(log.Info, "[Monitor Poller] Snapshot updated: %d sites, %d domains, %d endpoints, %d members offline",
		len(tmp.SiteResults), len(tmp.DomainResults), len(tmp.EndpointResults), len(offlineMembers))
}
