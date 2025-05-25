package main

import (
	"flag"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	api "ibp-geodns/src/serviceMonitor/api"
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

	// 1) Initialize config
	cfg.Init(*cfgFile)

	// 2) Parse configured log level
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// 3) Initialize data with local/official caching but NO usage stats
	dat.Init(dat.InitOptions{
		UseLocalOfficialCaches: true,
		UseUsageStats:          false,
	})

	// 4) Initialize MaxMind
	max.Init()

	// 5) Initialize data layer, load caches, etc. (We already did above + optional)
	//    We typically do official & local caches here, so it's done.

	// 6) Launch NATS
	nats.Init()

	// 7) Start monitor checks
	monitor.Init()

	// 8) Start the internal API to serve official results
	api.Init()

	// 9) Keep alive
	for {
		time.Sleep(60 * time.Second)
	}
}
