package webserver

import (
	"net/http"

	"ibp-geodns/src/common/config"
	l "ibp-geodns/src/common/logging"
	"ibp-geodns/src/powerdns-backend/networking/webserver/dns"
)

// Init initializes and starts the HTTP services for the PowerDNS backend.
// We now ONLY serve DNS queries. All management endpoints have been removed.
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

// handleDnsQuery delegates incoming requests to the DNS package logic.
func handleDnsQuery(w http.ResponseWriter, r *http.Request) {
	dns.DnsQueryHandler(w, r)
}
