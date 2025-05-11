package main

import (
	"flag"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	sig "ibp-geodns/src/common/signal"
	"ibp-geodns/src/dnsApi/api"
	mon "ibp-geodns/src/dnsApi/monitor"
)

var version = "0.3.0"

func main() {
	// Initialize the logging level
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "IBP-GeoDNS v%s starting...", version)

	// Define a command-line flag for the config file path
	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	// Check if the provided config file exists
	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Initialize Config file, memory pointers
	cfg.Init(*cfgFile)

	// Update maxmind, initialize geoip database
	max.Init()

	// Start data helper, Load caches
	dat.Init()

	// Sleep while we load caches
	time.Sleep(2 * time.Second)

	// Launch NATS Internode Communication
	sig.Init()

	// Launch RPC Monitor
	mon.Init()

	// Launch API Listener
	api.Init()

	// Infinite loop to keep things operational
	loop()
}

func loop() {
	for {
		time.Sleep(60 * time.Second)
	}
}
