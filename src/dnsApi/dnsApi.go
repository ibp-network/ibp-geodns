package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	api "ibp-geodns/src/dnsApi/api"
)

// We store a local snapshot of official results from the serviceMonitor
var (
	version                 = "0.4.0"
	officialResultsSnapshot OfficialResults
	resultsMu               sync.RWMutex
)

// OfficialResults struct for site/domain/endpoint statuses
type OfficialResults struct {
	SiteResults     []MonitorResultSite     `json:"SiteResults"`
	DomainResults   []MonitorResultDomain   `json:"DomainResults"`
	EndpointResults []MonitorResultEndpoint `json:"EndpointResults"`
}

// MonitorResultSite replicate essential structure for site checks
type MonitorResultSite struct {
	CheckName string                 `json:"CheckName"`
	Results   []MonitorResultGeneric `json:"Results"`
}

// MonitorResultDomain replicate domain checks
type MonitorResultDomain struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	Results   []MonitorResultGeneric `json:"Results"`
}

// MonitorResultEndpoint replicate endpoint checks
type MonitorResultEndpoint struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	RpcUrl    string                 `json:"RpcUrl"`
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultGeneric struct {
	MemberName string                 `json:"MemberName"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
}

// Poll the serviceMonitor for updated official results
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

	var tmp OfficialResults
	err = api.DecodeJSONBody(resp.Body, &tmp)
	if err != nil {
		log.Log(log.Warn, "Failed to decode official results: %v", err)
		return
	}

	resultsMu.Lock()
	officialResultsSnapshot = tmp
	resultsMu.Unlock()
	log.Log(log.Debug, "Updated local snapshot of official results.")
}

func main() {
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "IBP-GeoDNS DNS backend v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Initialize config
	cfg.Init(*cfgFile)

	// Initialize MaxMind
	max.Init()

	// Start polling serviceMonitor
	startMonitorPoller()

	// Start the DNS API
	api.Init()

	// Keep running
	for {
		time.Sleep(60 * time.Second)
	}
}
