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
	"ibp-geodns/src/serviceMonitor/api"
	"ibp-geodns/src/serviceMonitor/monitor"
)

var version = "1.0.0"

func main() {
	log.SetLogLevel(log.Info)
	log.Log(log.Info, "IBP-GeoDNS serviceMonitor v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// Initialize config
	cfg.Init(*cfgFile)

	// Initialize MaxMind
	max.Init()

	// Initialize data layer, load caches
	dat.Init()

	// small pause to ensure caches loaded
	time.Sleep(2 * time.Second)

	// Launch NATS
	nats.Init()

	// Start the monitor checks
	monitor.Init()

	// Start the internal API to serve official results or reset
	api.Init()

	// Keep alive
	for {
		time.Sleep(60 * time.Second)
	}
}
