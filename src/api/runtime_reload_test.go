package api

import (
	"testing"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
)

func TestSyncRuntimeConfigCachesRefreshesOnlyWhenInputsChange(t *testing.T) {
	prevGetConfig := getCurrentRuntimeConfig
	prevReadConfig := readRuntimeConfigFile
	prevStaticRefresh := refreshStaticDNSEntries
	prevServiceRefresh := refreshDynamicServiceRecords
	prevCountryRefresh := refreshCountryOverridesFromConfigFile
	prevDigest := lastRuntimeConfigDigest
	prevDigestLoaded := lastRuntimeConfigDigestLoaded
	t.Cleanup(func() {
		getCurrentRuntimeConfig = prevGetConfig
		readRuntimeConfigFile = prevReadConfig
		refreshStaticDNSEntries = prevStaticRefresh
		refreshDynamicServiceRecords = prevServiceRefresh
		refreshCountryOverridesFromConfigFile = prevCountryRefresh
		lastRuntimeConfigDigest = prevDigest
		lastRuntimeConfigDigestLoaded = prevDigestLoaded
	})

	lastRuntimeConfigDigest = [32]byte{}
	lastRuntimeConfigDigestLoaded = false

	currentConfig := cfg.Config{
		Services: map[string]cfg.Service{
			"rpc": {
				Configuration: cfg.ServiceConfiguration{Name: "RPC", Active: 1},
			},
		},
	}
	currentConfigFile := []byte(`{"GeoDNSOverrides":{"example.com":{"US":{"ipv4":"203.0.113.10"}}}}`)

	getCurrentRuntimeConfig = func() cfg.Config {
		return currentConfig
	}
	readRuntimeConfigFile = func() []byte {
		return append([]byte(nil), currentConfigFile...)
	}

	var staticCalls, serviceCalls, countryCalls int
	refreshStaticDNSEntries = func() { staticCalls++ }
	refreshDynamicServiceRecords = func() { serviceCalls++ }
	refreshCountryOverridesFromConfigFile = func() { countryCalls++ }

	syncRuntimeConfigCaches(false)
	syncRuntimeConfigCaches(false)

	if staticCalls != 1 || serviceCalls != 1 || countryCalls != 1 {
		t.Fatalf("expected one initial refresh, got static=%d service=%d country=%d", staticCalls, serviceCalls, countryCalls)
	}

	currentConfig.Services["rpc"] = cfg.Service{
		Configuration: cfg.ServiceConfiguration{Name: "RPC", Active: 0},
	}

	syncRuntimeConfigCaches(false)

	if staticCalls != 2 || serviceCalls != 2 || countryCalls != 2 {
		t.Fatalf("expected change-driven refresh, got static=%d service=%d country=%d", staticCalls, serviceCalls, countryCalls)
	}
}

func TestSyncRuntimeConfigCachesForceRefreshesEvenWithoutChanges(t *testing.T) {
	prevGetConfig := getCurrentRuntimeConfig
	prevReadConfig := readRuntimeConfigFile
	prevStaticRefresh := refreshStaticDNSEntries
	prevServiceRefresh := refreshDynamicServiceRecords
	prevCountryRefresh := refreshCountryOverridesFromConfigFile
	prevDigest := lastRuntimeConfigDigest
	prevDigestLoaded := lastRuntimeConfigDigestLoaded
	t.Cleanup(func() {
		getCurrentRuntimeConfig = prevGetConfig
		readRuntimeConfigFile = prevReadConfig
		refreshStaticDNSEntries = prevStaticRefresh
		refreshDynamicServiceRecords = prevServiceRefresh
		refreshCountryOverridesFromConfigFile = prevCountryRefresh
		lastRuntimeConfigDigest = prevDigest
		lastRuntimeConfigDigestLoaded = prevDigestLoaded
	})

	lastRuntimeConfigDigest = [32]byte{}
	lastRuntimeConfigDigestLoaded = false
	getCurrentRuntimeConfig = func() cfg.Config { return cfg.Config{} }
	readRuntimeConfigFile = func() []byte { return nil }

	var staticCalls int
	refreshStaticDNSEntries = func() { staticCalls++ }
	refreshDynamicServiceRecords = func() {}
	refreshCountryOverridesFromConfigFile = func() {}

	syncRuntimeConfigCaches(false)
	syncRuntimeConfigCaches(true)

	if staticCalls != 2 {
		t.Fatalf("expected forced sync to refresh twice total, got %d", staticCalls)
	}
}
