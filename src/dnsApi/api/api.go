package api

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"ibp-geodns/src/mgmtApi/api/types"
	"time"
)

var (
	ServiceRecords ServiceMap
	StaticRecords  StaticMap
	TLDRecords     TLDMap
)

// Init initializes the DNS server with the provided configuration.
func Init() {
	log.Log(log.Debug, "DNS Package initializing...")

	// Load staticEntries
	StaticDNSEntries()

	// Load unique topLevelDomains
	GenerateTLDs()

	// Load Dynamic Entries
	DynamicDNSEntries()

	// Start configuration updater
	go configUpdater()

	// Launch API Listener
	go Listener()
}

func configUpdater() {
	initialDelay := 2 * time.Second
	time.AfterFunc(initialDelay, configTimer)
}

func configTimer() {
	c := cfg.GetConfig()

	// Launch initial config update
	// Update Static Entries
	go StaticDNSEntries()
	// Update topLevelDomains
	go GenerateTLDs()
	// Load Dynamic Entries
	go DynamicDNSEntries()

	ticker := time.NewTicker(c.Local.System.ConfigReloadTime * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		log.Log(log.Debug, "Updating DNS configs")
		// Update Static Entries
		go StaticDNSEntries()
		// Update topLevelDomains
		go GenerateTLDs()
		// Load Dynamic Entries
		go DynamicDNSEntries()
	}
}

func Listener() {
	log.Log(log.Debug, "API Package initializing...")

	// Load configuration
	c := config.GetConfig()

	// DNS API for PowerDNS (unchanged)
	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", router)

	log.Log(log.Info, "Starting DNS API server on %s:%s",
		c.Local.DnsApi.ListenAddress,
		c.Local.DnsApi.ListenPort,
	)

	go http.ListenAndServe(
		c.Local.DnsApi.ListenAddress+":"+c.Local.DnsApi.ListenPort,
		dnsApi,
	)
}
