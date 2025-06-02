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

var version = "0.3.1"

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
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "serviceMonitor is running with log level: %s", c.Local.System.LogLevel)

	// 2) Initialize data
	dat.Init(dat.InitOptions{
		UseLocalOfficialCaches: true,
		UseUsageStats:          false,
	})

	// 3) Initialize MaxMind
	max.Init()

	// 4) Connect to NATS
	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	// 5) Set NodeID + ThisNode
	natsCommon.State.NodeID = c.Local.Nats.NodeID
	natsCommon.State.ThisNode = natsCommon.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPMonitor",
	}

	// 6) Enable Monitor role
	if err := natsCommon.EnableMonitorRole(); err != nil {
		log.Log(log.Fatal, "Failed to enable monitor role for NATS: %v", err)
		os.Exit(1)
	}

	// 7) Start checks
	monitor.Init()

	// 8) Start serviceMonitor API
	api.Init()

	// 9) Keep alive
	for {
		time.Sleep(60 * time.Second)
	}
}
