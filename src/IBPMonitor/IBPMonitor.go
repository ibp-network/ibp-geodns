package main

import (
	"flag"
	"os"
	"time"

	api "ibp-geodns/src/IBPMonitor/api"
	"ibp-geodns/src/IBPMonitor/monitor"
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	natsCommon "ibp-geodns/src/common/nats"
)

var version = "1.0.0"

func main() {
	log.Log(log.Info, "IBP-GeoDNS serviceMonitor v%s starting...", version)

	cfgFile := flag.String("config", "config.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// 1) Initialize config
	cfg.Init(*cfgFile)

	// 2) Parse configured log level from config
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "serviceMonitor is running with log level: %s", c.Local.System.LogLevel)

	// 3) Initialize data with local/official caching but NO usage stats
	dat.Init(dat.InitOptions{
		UseLocalOfficialCaches: true,
		UseUsageStats:          false,
	})

	// 4) Initialize MaxMind
	max.Init()

	// 5) Connect to NATS and enable Monitor role
	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}
	if err := natsCommon.EnableMonitorRole(); err != nil {
		log.Log(log.Fatal, "Failed to enable monitor role for NATS: %v", err)
		os.Exit(1)
	}

	// 6) Start monitor checks
	monitor.Init()

	// 7) Start the internal API to serve official results
	api.Init()

	// 8) Keep alive
	for {
		time.Sleep(60 * time.Second)
	}
}
