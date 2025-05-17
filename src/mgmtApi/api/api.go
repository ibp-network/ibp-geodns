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

	// Management API
	mgmtMux := http.NewServeMux()
	mgmtMux.HandleFunc("/api", router)

	log.Log(log.Info, "Starting Mgmt API server on %s:%s",
		c.Local.MgmtApi.ListenAddress,
		c.Local.MgmtApi.ListenPort,
	)

	go http.ListenAndServe(
		c.Local.MgmtApi.ListenAddress+":"+c.Local.MgmtApi.ListenPort,
		mgmtMux,
	)
}
