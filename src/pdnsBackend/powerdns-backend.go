package main

import (
	"flag"
	"os"
	"time"

	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data"
	l "ibp-geodns/src/common/logging"
	"ibp-geodns/src/pdnsBackend/dnsApi"
	"ibp-geodns/src/pdnsBackend/rpcMonitor"
)

var version = "0.2.0"

func main() {
	// Initialize the logging level
	l.SetLogLevel(l.Debug)
	l.Log(l.Info, "IBP-GeoDNS v%s starting...", version)

	// Define a command-line flag for the config file path
	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	// Check if the provided config file exists
	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		l.Log(l.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Initialize components
	config.Init(*cfgFile)

	// Start data package
	data.Init()

	// Sleep to load the caches load
	time.Sleep(1 * time.Second)

	// Launch RPC Monitor
	rpcMonitor.Init()

	// Launch DNS / PDNS API
	dnsApi.Init()

	// Infinite loop to keep things operational
	loop()
}

func loop() {
	for {
		time.Sleep(60 * time.Second)
	}
}
