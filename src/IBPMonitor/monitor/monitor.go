package monitor

import (
	log "ibp-geodns/src/common/logging"
)

var manager *CheckManager

func Init() {
	log.Log(log.Debug, "Monitor Package initializing...")

	// Create and start the check manager
	manager = NewCheckManager()
	manager.Start()
}

func Shutdown() {
	if manager != nil {
		manager.Stop()
	}
}
