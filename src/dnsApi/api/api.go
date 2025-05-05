package dnsApi

import (
	"net/http"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

func Init() {
	log.Log(log.Debug, "API Package initializing...")

	// Load configuration
	c := cfg.GetConfig()

	// DNS API for PowerDNS (unchanged)
	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", handleDnsQuery)

	log.Log(log.Info, "Starting DNS API server on %s:%s",
		c.System.DnsApi.ListenAddress,
		c.System.DnsApi.ListenPort,
	)

	go http.ListenAndServe(
		c.System.DnsApi.ListenAddress+":"+c.System.DnsApi.ListenPort,
		dnsApi,
	)
}
