package webserver

import (
	"ibp-geodns/src/common/config"
	l "ibp-geodns/src/common/logging"
	"ibp-geodns/src/powerdns-backend/networking/webserver/api"
	"ibp-geodns/src/powerdns-backend/networking/webserver/dns"
	"net/http"
)

func Init() {
	l.Log(l.Debug, "API Package initializing...")
	c := config.GetConfig()

	// Launch Init functions of sub-packages
	go api.Init()
	go dns.Init()

	// Define DNS API
	dnsApi := http.NewServeMux()
	dnsApi.HandleFunc("/dns", handleDnsQuery)

	// Launch DNS API Thread
	l.Log(l.Info, "Starting DNS API server on %s:%s", c.System.DnsApi.ListenAddress, c.System.DnsApi.ListenPort)
	go http.ListenAndServe(c.System.DnsApi.ListenAddress+":"+c.System.DnsApi.ListenPort, dnsApi)

	// Define Management API
	mgmtApi := http.NewServeMux()
	mgmtApi.HandleFunc("/api/billing", handleApiQuery)
	mgmtApi.HandleFunc("/api/member", handleApiQuery)
	mgmtApi.HandleFunc("/api/status", handleApiQuery)
	mgmtApi.HandleFunc("/api/usage", handleApiQuery)

	// Launch MGMT API Thread
	l.Log(l.Info, "Starting MGMT API server on %s:%s", c.System.MgmtApi.ListenAddress, c.System.MgmtApi.ListenPort)
	go http.ListenAndServe(c.System.MgmtApi.ListenAddress+":"+c.System.MgmtApi.ListenPort, mgmtApi)
}
