package api

import (
	"strings"
	"sync"
)

var (
	runtimeConfigMu   sync.RWMutex
	runtimeConfigPath = "ibpdns.json"
)

// SetConfigPath records the main config file path so auxiliary loaders can
// read extensions that are not exposed by the shared config library.
func SetConfigPath(path string) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return
	}

	runtimeConfigMu.Lock()
	runtimeConfigPath = trimmed
	runtimeConfigMu.Unlock()
}

func getConfigPath() string {
	runtimeConfigMu.RLock()
	defer runtimeConfigMu.RUnlock()
	return runtimeConfigPath
}
