package api

import (
	"ibp-geodns/src/common/config"
	l "ibp-geodns/src/common/logging"
	"time"
)

var (
	ServiceRecords ServiceMap
	StaticRecords  StaticMap
	TLDRecords     TLDMap
)

// Init initializes the DNS server with the provided configuration.
func Init() {
	l.Log(l.Debug, "DNS Package initializing...")

	// Load staticEntries
	StaticDNSEntries()

	// Load unique topLevelDomains
	GenerateTLDs()

	// Load Dynamic Entries
	DynamicDNSEntries()

	// Start configuration updater
	go configUpdater()
}

func configUpdater() {
	initialDelay := 2 * time.Second
	time.AfterFunc(initialDelay, configTimer)
}

func configTimer() {
	c := config.GetConfig()

	// Launch initial config update
	// Update Static Entries
	go StaticDNSEntries()
	// Update topLevelDomains
	go GenerateTLDs()
	// Load Dynamic Entries
	go DynamicDNSEntries()

	ticker := time.NewTicker(c.System.ConfigReloadTime * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		l.Log(l.Debug, "Updating DNS configs")
		// Update Static Entries
		go StaticDNSEntries()
		// Update topLevelDomains
		go GenerateTLDs()
		// Load Dynamic Entries
		go DynamicDNSEntries()
	}
}
