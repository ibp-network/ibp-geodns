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

var version = "0.4.1"

func main() {
	log.Log(log.Info, "IBP‑GeoDNS serviceMonitor v%s starting …", version)

	cfgPath := flag.String("config", "ibpmonitor.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgPath); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgPath)
		os.Exit(1)
	}

	//-----------------------------------------------------------------------
	// bootstrap subsystems
	//-----------------------------------------------------------------------

	cfg.Init(*cfgPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	dat.Init(dat.InitOptions{UseLocalOfficialCaches: true, UseUsageStats: false})
	max.Init()

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	//-----------------------------------------------------------------------
	// advertise ourselves (IBPMonitor role)
	//-----------------------------------------------------------------------

	natsCommon.State.NodeID = c.Local.Nats.NodeID
	natsCommon.State.ThisNode = natsCommon.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPMonitor",
	}

	if err := natsCommon.EnableMonitorRole(); err != nil {
		log.Log(log.Fatal, "Failed to enable monitor role for NATS: %v", err)
		os.Exit(1)
	}

	//-----------------------------------------------------------------------
	// start health‑checks & HTTP API
	//-----------------------------------------------------------------------

	monitor.Init()
	api.Init()

	//-----------------------------------------------------------------------
	// run forever
	//-----------------------------------------------------------------------

	for {
		time.Sleep(60 * time.Second)
	}
}
