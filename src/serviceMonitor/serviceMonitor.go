package main

import (
	"flag"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"ibp-geodns/src/serviceMonitor/api"
	"ibp-geodns/src/serviceMonitor/monitor"
	nats "ibp-geodns/src/serviceMonitor/nats"
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

	// Load config file and set log level fromt he config file
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// Initialize MaxMind
	max.Init()

	// Initialize data layer, load caches
	dat.Init()

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
