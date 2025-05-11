package api

import (
	"net/http"

	"ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

func Init() {
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
