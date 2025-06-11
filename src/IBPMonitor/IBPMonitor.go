package main

import (
	"flag"
	"os"
	"time"

	api "ibp-geodns/src/IBPMonitor/api"
	ibpcons "ibp-geodns/src/IBPMonitor/consensus"
	"ibp-geodns/src/IBPMonitor/monitor"
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	natsCommon "ibp-geodns/src/common/nats"
)

var version = "0.4.0"

func main() {
	log.Log(log.Info, "IBP‑GeoDNS serviceMonitor v%s starting ...", version)

	cfgFile := flag.String("config", "ibpmonitor.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// bootstrap subsystems
	// -----------------------------------------------------------------------
	cfg.Init(*cfgFile)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	dat.Init(dat.InitOptions{UseLocalOfficialCaches: true, UseUsageStats: false})
	max.Init()

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	// -----------------------------------------------------------------------
	// consensus manager  (NEW)
	// -----------------------------------------------------------------------
	nc := natsCommon.GetConnection()
	consMgr := ibpcons.NewManager(c.Local.Nats.NodeID, nc, 1) // quorum re‑computed later
	monitor.SetConsensusManager(consMgr)

	// -----------------------------------------------------------------------
	// existing NATS heartbeat/roles (kept for metrics)
	// -----------------------------------------------------------------------
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

	// -----------------------------------------------------------------------
	// start checks + API
	// -----------------------------------------------------------------------
	monitor.Init()
	api.Init()

	for {
		time.Sleep(60 * time.Second)
	}
}
