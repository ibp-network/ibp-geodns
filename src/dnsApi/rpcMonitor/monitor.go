package rpcMonitor

import (
	log "ibp-geodns/src/common/logging"
)

func Init() {
	log.Log(log.Debug, "Monitor Package initializing...")

	// Start checks
	go startChecks()
}
