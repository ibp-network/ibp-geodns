package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	nconn "ibp-geodns/src/common/nats"
)

var version = "0.7.0"

func main() {
	log.Log(log.Info, "IBP-GeoDNS Collator v%s starting...", version)

	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Printf("Configuration file not found: %s\n", *configPath)
		os.Exit(1)
	}

	// 1) Load config
	cfg.Init(*configPath)
	c := cfg.GetConfig()

	// 2) Set logging level
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "[Collator] Starting with config=%s", *configPath)

	// 3) Connect to NATS
	err := nconn.Connect()
	if err != nil {
		log.Log(log.Fatal, "[Collator] Failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	// 4) Make sure we set NodeID/ThisNode BEFORE enabling the role
	nconn.State.NodeID = c.Local.Nats.NodeID
	nconn.State.ThisNode = nconn.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPCollator",
	}

	// 5) Enable Collator role
	err = nconn.EnableCollatorRole()
	if err != nil {
		log.Log(log.Fatal, "[Collator] Failed to enable Collator role: %v", err)
		os.Exit(1)
	}

	// 6) Wait for membership if desired (DNS + Monitor):
	log.Log(log.Info, "[Collator] Waiting for IBPDns membership...")
	foundDns := nconn.WaitForNodesByRole("IBPDns", 1, 5*time.Second)
	if !foundDns {
		log.Log(log.Warn, "[Collator] No IBPDns found within 5s; continuing anyway.")
	}
	log.Log(log.Info, "[Collator] Waiting for IBPMonitor membership...")
	foundMon := nconn.WaitForNodesByRole("IBPMonitor", 1, 5*time.Second)
	if !foundMon {
		log.Log(log.Warn, "[Collator] No IBPMonitor found within 5s; continuing anyway.")
	}

	// 7) Make usage request
	log.Log(log.Info, "[Collator] Requesting usage data from all IBPDns nodes...")
	usageReq := nconn.UsageRequest{
		StartDate:  "2025-05-30",
		EndDate:    "2025-05-30",
		Domain:     "",
		MemberName: "",
		Country:    "",
	}
	usageTimeout := 5 * time.Second
	usageRecords, err := nconn.RequestAllDnsUsage(usageReq, usageTimeout)
	if err != nil {
		log.Log(log.Error, "[Collator] Usage request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d usage records", len(usageRecords))
	}

	// 8) Make downtime request
	log.Log(log.Info, "[Collator] Requesting downtime data from all IBPMonitor nodes...")
	dtReq := nconn.DowntimeRequest{
		StartTime:  time.Date(2025, 05, 30, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2025, 05, 30, 23, 59, 59, 0, time.UTC),
		MemberName: "",
	}
	dtTimeout := 5 * time.Second
	dtEvents, err := nconn.RequestAllMonitorsDowntime(dtReq, dtTimeout)
	if err != nil {
		log.Log(log.Error, "[Collator] Downtime request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d downtime events", len(dtEvents))
	}

	// 9) Print usage + downtime
	fmt.Printf("=== Usage Records (count=%d) ===\n", len(usageRecords))
	for i, rec := range usageRecords {
		fmt.Printf("[%d] Date=%s Domain=%s Member=%s Country=%s Hits=%d\n",
			i+1, rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
	}

	fmt.Printf("\n=== Downtime Events (count=%d) ===\n", len(dtEvents))
	for i, evt := range dtEvents {
		fmt.Printf("[%d] Member=%s CheckType=%s CheckName=%s Domain=%s Endpoint=%s Status=%v Start=%s End=%s\n",
			i+1, evt.MemberName, evt.CheckType, evt.CheckName, evt.DomainName, evt.Endpoint, evt.Status,
			evt.StartTime.Format(time.RFC3339), evt.EndTime.Format(time.RFC3339))
	}

	log.Log(log.Info, "[Collator] Done, exiting now.")

	time.Sleep(1 * time.Second)
	nconn.Disconnect()
}
