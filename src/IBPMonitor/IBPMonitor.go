package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "ibp-geodns/src/IBPMonitor/api"
	"ibp-geodns/src/IBPMonitor/monitor"
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	natsCommon "ibp-geodns/src/common/nats"
)

var version = cfg.GetVersion()

func main() {
	log.Log(log.Info, "IBPMonitor %s starting...", version)

	cfgPath := flag.String("config", "ibpmonitor.json", "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgPath); os.IsNotExist(err) {
		log.Log(log.Fatal, "Configuration file not found: %s", *cfgPath)
		os.Exit(1)
	}

	cfg.Init(*cfgPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	dat.Init(dat.InitOptions{UseLocalOfficialCaches: true, UseUsageStats: false})
	max.Init()

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

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

	monitor.Init()
	api.Init()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Log(log.Info, "Shutdown signal received, cleaning up...")
	monitor.Shutdown()
	time.Sleep(1 * time.Second) // Give time for cleanup
}
