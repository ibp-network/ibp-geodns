package monitor

import (
	l "ibp-geodns/src/common/logging"
)

func Init() {
	l.Log(l.Debug, "Monitor Package initializing...")

	// Start checks
	go startChecks()
}
