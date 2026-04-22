package api

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"sync"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

const runtimeConfigRefreshCheckInterval = 30 * time.Second

var (
	getCurrentRuntimeConfig = cfg.GetConfig
	readRuntimeConfigFile   = func() []byte {
		data, err := os.ReadFile(getConfigPath())
		if err != nil {
			return nil
		}
		return data
	}
	refreshStaticDNSEntries               = StaticDNSEntries
	refreshDynamicServiceRecords          = RebuildServiceRecords
	refreshCountryOverridesFromConfigFile = LoadCountryOverridesFromConfigFile

	runtimeConfigStateMu          sync.Mutex
	lastRuntimeConfigDigest       [sha256.Size]byte
	lastRuntimeConfigDigestLoaded bool
	runtimeConfigWatcherOnce      sync.Once
)

func runtimeConfigDigest() ([sha256.Size]byte, error) {
	currentConfig := getCurrentRuntimeConfig()
	snapshot := struct {
		StaticDNS  []cfg.DNSRecord        `json:"staticDNS"`
		Members    map[string]cfg.Member  `json:"members"`
		Services   map[string]cfg.Service `json:"services"`
		ConfigFile []byte                 `json:"configFile"`
	}{
		StaticDNS:  currentConfig.StaticDNS,
		Members:    currentConfig.Members,
		Services:   currentConfig.Services,
		ConfigFile: readRuntimeConfigFile(),
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return [sha256.Size]byte{}, err
	}

	return sha256.Sum256(payload), nil
}

func syncRuntimeConfigCaches(force bool) {
	digest, err := runtimeConfigDigest()
	if err != nil {
		log.Log(log.Warn, "syncRuntimeConfigCaches: failed to compute runtime config digest: %v", err)
		return
	}

	runtimeConfigStateMu.Lock()
	changed := force || !lastRuntimeConfigDigestLoaded || digest != lastRuntimeConfigDigest
	if changed {
		lastRuntimeConfigDigest = digest
		lastRuntimeConfigDigestLoaded = true
	}
	runtimeConfigStateMu.Unlock()

	if !changed {
		return
	}

	log.Log(log.Info, "syncRuntimeConfigCaches: syncing DNS runtime caches from latest config")
	refreshStaticDNSEntries()
	refreshDynamicServiceRecords()
	refreshCountryOverridesFromConfigFile()
}

func startRuntimeConfigWatcher() {
	runtimeConfigWatcherOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(runtimeConfigRefreshCheckInterval)
			defer ticker.Stop()

			for range ticker.C {
				syncRuntimeConfigCaches(false)
			}
		}()
	})
}
