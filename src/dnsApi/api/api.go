package dnsApi

import (
	"net/http"

	"ibp-geodns/src/common/config"
	l "ibp-geodns/src/common/logging"
)

func Init() {
	l.Log(l.Debug, "API Package initializing...")

	// Load configuration
	c := config.GetConfig()

	// DNS API for PowerDNS (unchanged)
	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", handleDnsQuery)

	l.Log(l.Info, "Starting DNS API server on %s:%s",
		c.System.DnsApi.ListenAddress,
		c.System.DnsApi.ListenPort,
	)

	go http.ListenAndServe(
		c.System.DnsApi.ListenAddress+":"+c.System.DnsApi.ListenPort,
		dnsApi,
	)
}
