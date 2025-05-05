package config

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"runtime"
	"time"

	log "ibp-geodns/src/common/logging"
)

var (
	cfg *Config
)

// NewConfig creates a new Config instance and starts the update ticker
func Init(cfgFile string) {
	log.Log(log.Debug, "Config Package initializing...")
	cfg = &Config{
		cfgFile: cfgFile,
	}

	loadConfig(cfgFile, true)
	go configUpdater(cfgFile)
}

// loadConfig loads all configuration files
func loadConfig(cfgFile string, initialLoad bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// Load system config from disk
	loadSystemConfig(cfgFile, initialLoad)

	// Load other configs from URLs
	loadStaticDNSConfig(cfg.data.System.StaticDNSConfig, initialLoad)
	loadMembersConfig(cfg.data.System.MembersConfig, initialLoad)
	loadServicesConfig(cfg.data.System.ServicesConfig, initialLoad)
	loadIaasPricing(cfg.data.System.IaasPricingConfig, initialLoad)
	loadServiceRequestsConfig(cfg.data.System.ServicesRequestsConfig, initialLoad)
}

// loadSystemConfig loads the system config from disk
func loadSystemConfig(configPath string, initialLoad bool) {
	file, err := os.Open(configPath)
	if err != nil {
		log.Log(log.Error, "Failed to open system config file: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}
	defer file.Close()

	var systemConfig SystemConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&systemConfig); err != nil {
		log.Log(log.Error, "Failed to decode system config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.System = systemConfig
	log.Log(log.Debug, "System configuration loaded from %s", configPath)
}

// loadStaticDNSConfig loads the static DNS config from a URL
func loadStaticDNSConfig(url string, initialLoad bool) {
	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}

	var records []DNSRecord
	if err := json.Unmarshal(data, &records); err != nil {
		log.Log(log.Error, "Failed to unmarshal StaticDNS config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.StaticDNS = records
	log.Log(log.Debug, "StaticDNS configuration loaded from %s", url)
}

// loadMembersConfig loads the members config from a URlog.
func loadMembersConfig(url string, initialLoad bool) {
	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}

	var newMembers map[string]Member
	if err := json.Unmarshal(data, &newMembers); err != nil {
		log.Log(log.Error, "Failed to unmarshal Members config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	// Retain Override = 1 for existing members in cfg.data.Members
	for name, existingMember := range cfg.data.Members {
		if existingMember.Override {
			if newMember, exists := newMembers[name]; exists {
				newMember.Override = true
				newMembers[name] = newMember
			}
		}
	}

	// Overwrite existing members with the new configuration
	cfg.data.Members = newMembers
	log.Log(log.Debug, "Members configuration loaded from %s", url)
}

// loadServicesConfig loads the services config from a URL
func loadServicesConfig(url string, initialLoad bool) {
	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}
	var services map[string]Service
	if err := json.Unmarshal(data, &services); err != nil {
		log.Log(log.Error, "Failed to unmarshal Services config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.Services = services
	log.Log(log.Debug, "Services configuration loaded from %s", url)
}

// loadIaasPricing loads the IaaS Pricing data from a URL
func loadIaasPricing(url string, initialLoad bool) {
	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}

	var pricing map[string]IaasPricing
	if err := json.Unmarshal(data, &pricing); err != nil {
		log.Log(log.Error, "Failed to unmarshal IaaS pricing config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.Pricing = pricing
	log.Log(log.Debug, "IaaS pricing configuration loaded from %s", url)
}

// loadSaasPricing loads the SASS Pricing data from a URL
func loadServiceRequestsConfig(url string, initialLoad bool) {
	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}
	var requests ServiceRequests
	if err := json.Unmarshal(data, &requests); err != nil {
		_, _, line, _ := runtime.Caller(2)
		log.Log(log.Error, "Failed to unmarshal Services config Line: %d Error: %v", line, err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.ServiceRequests = requests
	log.Log(log.Debug, "Services configuration loaded from %s", url)
}

// downloadConfig downloads a config file from a URL
func downloadConfig(url string, initialLoad bool) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Log(log.Error, "Failed to download config from %s: %v", url, err)
		if initialLoad {
			_, _, line, _ := runtime.Caller(2)
			log.Log(log.Fatal, "Terminating program due to critical error on initial load. Line: %d", line)
			os.Exit(1)
		}
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Log(log.Error, "Non-OK HTTP status while downloading config from %s: %s", url, resp.Status)
		if initialLoad {
			_, _, line, _ := runtime.Caller(2)
			log.Log(log.Fatal, "Terminating program due to critical error on initial load. Line: %d", line)
			os.Exit(1)
		}
		return nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log(log.Error, "Failed to read response body from %s: %v", url, err)
		if initialLoad {
			_, _, line, _ := runtime.Caller(2)
			log.Log(log.Fatal, "Terminating program due to critical error on initial load. Line: %d", line)
			os.Exit(1)
		}
		return nil
	}

	return data
}

// startTicker starts a ticker to periodically update the configs
func configUpdater(cfgFile string) {
	c := GetConfig()

	ticker := time.NewTicker(c.System.ConfigReloadTime * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		loadConfig(cfgFile, false)
	}
}

// GetConfig returns a deep copy of the current configuration data
func GetConfig() ConfigData {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	// Deep copy using JSON marshal and unmarshal
	var dataCopy ConfigData
	dataBytes, err := json.Marshal(cfg.data)
	if err != nil {
		log.Log(log.Error, "Failed to marshal configuration data: %v", err)
	} else {
		err = json.Unmarshal(dataBytes, &dataCopy)
		if err != nil {
			log.Log(log.Error, "Failed to unmarshal configuration data: %v", err)
		}
	}

	return dataCopy
}
