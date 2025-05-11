package main

import (
	"flag"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	nats "ibp-geodns/src/common/nats"
	api "ibp-geodns/src/dnsApi/api"
)

var version = "0.2.0"

// main starts your standalone mgmt-api server.
func main() {
	// Initialize the logging level
	log.SetLogLevel(log.Debug)
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
	nats.Init()

	// Launch API Listener
	api.Init()

	go loop()
}

func loop() {
	for {
		time.Sleep(60 * time.Second)
	}
}
