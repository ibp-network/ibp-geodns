package main

import (
	"flag"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
	nats "ibp-geodns/src/common/nats"
)

var version = cfg.GetVersion()

func main() {
	log.Log(log.Info, "IBPCollator v%s starting …", version)

	cfgPath := flag.String("config", "ibpcollator.json", "Path to configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgPath); os.IsNotExist(err) {
		log.Log(log.Fatal, "configuration file not found: %s", *cfgPath)
		os.Exit(1)
	}

	cfg.Init(*cfgPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	go data2.Init()

	if err := nats.Connect(); err != nil {
		log.Log(log.Fatal, "NATS connect: %v", err)
		os.Exit(1)
	}

	nats.State.NodeID = c.Local.Nats.NodeID
	nats.State.ThisNode = nats.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPCollator",
	}
	if err := nats.EnableCollatorRole(); err != nil {
		log.Log(log.Fatal, "enable collator role: %v", err)
		os.Exit(1)
	}

	if err := nats.StartCollatorServices(); err != nil {
		log.Log(log.Fatal, "collator services: %v", err)
		os.Exit(1)
	}

	log.Log(log.Info, "[collator] started – awaiting events")
	for {
		time.Sleep(1 * time.Hour)
	}
}
