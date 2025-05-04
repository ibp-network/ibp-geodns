package monitor

import (
	l "common/logging"
)

func Init() {
	l.Log(l.Debug, "Monitor Package initializing...")

	// Start checks
	go startChecks()
}
