package api

import (
	"net/http"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

func Init() {
	log.Log(log.Info, "DNS Package initializing...")

	c := cfg.GetConfig()

	//    e.g. StaticDNSEntries() is your existing function that sets StaticRecords
	StaticDNSEntries()

	// 2) Now that we have static records, gather TLDs
	populateTLDRecords()

	// The DNS service for PowerDNS
	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", dnsApiRouter)

	log.Log(log.Info, "Starting DNS API server on %s:%s",
		c.Local.DnsApi.ListenAddress,
		c.Local.DnsApi.ListenPort,
	)

	go http.ListenAndServe(
		c.Local.DnsApi.ListenAddress+":"+c.Local.DnsApi.ListenPort,
		dnsApi,
	)
}
