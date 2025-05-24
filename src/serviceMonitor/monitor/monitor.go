package monitor

import (
	log "ibp-geodns/src/common/logging"
)

// Init starts all checks
func Init() {
	log.Log(log.Debug, "Monitor Package initializing...")
	startChecks()
}
