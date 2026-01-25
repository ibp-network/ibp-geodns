package api

import (
	"encoding/json"
	"os"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
	natsCommon "github.com/ibp-network/ibp-geodns-libs/nats"
)

// LoadCountryOverridesFromConfigFile loads country overrides from the config file
// Expects a "GeoDNSOverrides" section in the config JSON
func LoadCountryOverridesFromConfigFile() {
	c := cfg.GetConfig()

	// Try to read the config file directly to get GeoDNSOverrides
	// Since the config library may not expose this field, we'll read the file
	var configPath string
	workDir := c.Local.System.WorkDir
	if workDir != "" && workDir[len(workDir)-1] != '/' {
		workDir += "/"
	}
	configPath = workDir + "ibpdns.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try current directory
		configPath = "ibpdns.json"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// Try config directory
			configPath = "config/ibpdns.json"
		}
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Log(log.Debug, "LoadCountryOverridesFromConfigFile: could not read config file: %v", err)
		return
	}

	var configJSON map[string]interface{}
	if err := json.Unmarshal(configData, &configJSON); err != nil {
		log.Log(log.Debug, "LoadCountryOverridesFromConfigFile: could not parse config JSON: %v", err)
		return
	}

	geoDNSOverridesRaw, exists := configJSON["GeoDNSOverrides"]
	if !exists {
		log.Log(log.Debug, "LoadCountryOverridesFromConfigFile: no GeoDNSOverrides section found in config")
		return
	}

	// Convert to the expected structure
	overridesJSON, err := json.Marshal(geoDNSOverridesRaw)
	if err != nil {
		log.Log(log.Warn, "LoadCountryOverridesFromConfigFile: could not marshal overrides: %v", err)
		return
	}

	var overrides map[string]map[string]CountryOverride
	if err := json.Unmarshal(overridesJSON, &overrides); err != nil {
		log.Log(log.Warn, "LoadCountryOverridesFromConfigFile: could not unmarshal overrides: %v", err)
		return
	}

	LoadCountryOverridesFromConfig(overrides)
}

// setupNatsCountryOverrideHandler sets up a NATS subscription for runtime country override updates
func setupNatsCountryOverrideHandler() {
	// Retry until NATS is available to avoid missing runtime overrides
	go func() {
		for {
			if natsCommon.NC == nil {
				log.Log(log.Debug, "setupNatsCountryOverrideHandler: NATS not connected, retrying in 5s")
				time.Sleep(5 * time.Second)
				continue
			}

			subject := "geodns.override.update"
			sub, err := natsCommon.NC.Subscribe(subject, func(msg *natsCommon.NatsMsg) {
				if msg != nil && len(msg.Data) > 0 {
					handleNatsCountryOverrideUpdate(msg.Data)
				}
			})

			if err != nil {
				log.Log(log.Warn, "setupNatsCountryOverrideHandler: failed to subscribe to %s: %v; retrying in 5s", subject, err)
				time.Sleep(5 * time.Second)
				continue
			}

			if sub != nil {
				log.Log(log.Info, "setupNatsCountryOverrideHandler: subscribed to NATS subject: %s", subject)
			}
			return
		}
	}()
}

// handleNatsCountryOverrideUpdate processes NATS messages for country override updates
// Expected message format:
//
//	{
//	  "action": "set" | "remove" | "clear",
//	  "domain": "example.com",
//	  "countryCode": "CN",
//	  "override": { "memberName": "...", "ipv4": "...", "ipv6": "..." }  // only for "set"
//	}
func handleNatsCountryOverrideUpdate(data []byte) {
	var msg struct {
		Action      string          `json:"action"`
		Domain      string          `json:"domain"`
		CountryCode string          `json:"countryCode"`
		Override    CountryOverride `json:"override,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		log.Log(log.Warn, "handleNatsCountryOverrideUpdate: failed to unmarshal message: %v", err)
		return
	}

	switch msg.Action {
	case "set":
		if msg.Domain == "" || msg.CountryCode == "" {
			log.Log(log.Warn, "handleNatsCountryOverrideUpdate: missing domain or countryCode for set action")
			return
		}
		CountryOverrides.SetCountryOverride(msg.Domain, msg.CountryCode, msg.Override)

	case "remove":
		if msg.Domain == "" || msg.CountryCode == "" {
			log.Log(log.Warn, "handleNatsCountryOverrideUpdate: missing domain or countryCode for remove action")
			return
		}
		CountryOverrides.RemoveCountryOverride(msg.Domain, msg.CountryCode)

	case "clear":
		if msg.Domain == "" {
			log.Log(log.Warn, "handleNatsCountryOverrideUpdate: missing domain for clear action")
			return
		}
		// Clear all overrides for a domain
		CountryOverrides.mu.Lock()
		delete(CountryOverrides.overrides, msg.Domain)
		CountryOverrides.mu.Unlock()
		log.Log(log.Info, "handleNatsCountryOverrideUpdate: cleared all overrides for domain=%s", msg.Domain)

	default:
		log.Log(log.Warn, "handleNatsCountryOverrideUpdate: unknown action: %s", msg.Action)
	}
}
