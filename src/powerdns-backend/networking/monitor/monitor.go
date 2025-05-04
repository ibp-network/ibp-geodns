package monitor

import (
	l "ibp-geodns/logging"
)

func Init() {
	l.Log(l.Debug, "Monitor Package initializing...")

	// Start checks
	go startChecks()
}
