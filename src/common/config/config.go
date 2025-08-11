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
	cfg *ConfigInit
)

func Init(cfgFile string) {
	log.Log(log.Debug, "Config Package initializing...")
	cfg = &ConfigInit{
		cfgFile: cfgFile,
	}

	loadConfig(cfgFile, true)
	go configUpdater(cfgFile)
}

func loadConfig(cfgFile string, initialLoad bool) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	loadSystemConfig(cfgFile, initialLoad)

	loadStaticDNSConfig(cfg.data.Local.System.ConfigUrls.StaticDNSConfig, initialLoad)
	loadMembersConfig(cfg.data.Local.System.ConfigUrls.MembersConfig, initialLoad)
	loadServicesConfig(cfg.data.Local.System.ConfigUrls.ServicesConfig, initialLoad)
	loadIaasPricing(cfg.data.Local.System.ConfigUrls.IaasPricingConfig, initialLoad)
	loadServiceRequestsConfig(cfg.data.Local.System.ConfigUrls.ServicesRequestsConfig, initialLoad)
	loadAlertsConfig(cfg.data.Local.System.ConfigUrls.AlertsConfig, initialLoad)
}

func loadAlertsConfig(url string, initialLoad bool) {
	if url == "" {
		// default to hardcoded URL if not specified
		url = "https://raw.githubusercontent.com/ibp-network/config/refs/heads/main/alerts.json"
	}

	data := downloadConfig(url, initialLoad)
	if data == nil {
		return
	}

	var alerts AlertsConfig
	if err := json.Unmarshal(data, &alerts); err != nil {
		log.Log(log.Error, "Failed to unmarshal Alerts config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.Alerts = alerts
	log.Log(log.Debug, "Alerts configuration loaded from %s", url)
}

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

	var systemConfig LocalConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&systemConfig); err != nil {
		log.Log(log.Error, "Failed to decode system config: %v", err)
		if initialLoad {
			log.Log(log.Fatal, "Terminating program due to critical error on initial load.")
			os.Exit(1)
		}
		return
	}

	cfg.data.Local = systemConfig
	log.Log(log.Debug, "System configuration loaded from %s", configPath)
}

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

	for name, existingMember := range cfg.data.Members {
		if existingMember.Override {
			if newMember, exists := newMembers[name]; exists {
				newMember.Override = true
				newMember.OverrideTime = existingMember.OverrideTime
				newMembers[name] = newMember
			}
		}
	}

	cfg.data.Members = newMembers
	log.Log(log.Debug, "Members configuration loaded from %s", url)
}

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

func downloadConfig(url string, initialLoad bool) []byte {
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Log(log.Error, "Failed to create HTTP request for config from %s: %v", url, err)
		if initialLoad {
			_, _, line, _ := runtime.Caller(2)
			log.Log(log.Fatal, "Terminating program due to critical error on initial load. Line: %d", line)
			os.Exit(1)
		}
		return nil
	}

	resp, err := client.Do(req)
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

func configUpdater(cfgFile string) {
	c := GetConfig()

	ticker := time.NewTicker(c.Local.System.ConfigReloadTime * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		loadConfig(cfgFile, false)
	}
}

func GetConfig() Config {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()

	var dataCopy Config
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
