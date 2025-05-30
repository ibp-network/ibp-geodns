package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	// Our shared packages
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	nconn "ibp-geodns/src/common/nats"
)

func main() {
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Printf("Configuration file not found: %s\n", *configPath)
		os.Exit(1)
	}

	cfg.Init(*configPath)
	c := cfg.GetConfig()

	// Set logging level
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "[Collator] Starting with config=%s", *configPath)

	// Connect to NATS
	err := nconn.Connect()
	if err != nil {
		log.Log(log.Fatal, "[Collator] Failed to connect to NATS: %v", err)
		os.Exit(1)
	}
	defer nconn.Disconnect()

	// -- IMPORTANT: Set NodeID and ThisNode BEFORE enabling the role:
	nconn.State.NodeID = c.Local.Nats.NodeID
	nconn.State.ThisNode = nconn.NodeInfo{
		NodeID: c.Local.Nats.NodeID,
		// Fill in other fields if desired, e.g. ListenAddress, PublicAddress, etc.
	}

	// Now enable Collator role
	err = nconn.EnableCollatorRole()
	if err != nil {
		log.Log(log.Fatal, "[Collator] Failed to enable Collator role: %v", err)
		os.Exit(1)
	}

	// 1) Request usage from DNS
	log.Log(log.Info, "[Collator] Requesting usage data from all IBPDns nodes...")

	usageReq := nconn.UsageRequest{
		StartDate:  "2025-05-30",
		EndDate:    "2025-05-30",
		Domain:     "",
		MemberName: "",
		Country:    "",
	}
	usageRecords, err := nconn.RequestAllDnsUsage(usageReq, 5*time.Second)
	if err != nil {
		log.Log(log.Error, "[Collator] Usage request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d usage records", len(usageRecords))
	}

	// 2) Request downtime from all IBPMonitor nodes
	log.Log(log.Info, "[Collator] Requesting downtime data from all IBPMonitor nodes...")
	dtReq := nconn.DowntimeRequest{
		StartTime:  time.Date(2025, 05, 30, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2025, 05, 30, 23, 59, 59, 0, time.UTC),
		MemberName: "",
	}
	dtEvents, err := nconn.RequestAllMonitorsDowntime(dtReq, 5*time.Second)
	if err != nil {
		log.Log(log.Error, "[Collator] Downtime request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d downtime events", len(dtEvents))
	}

	// Print usage/downtime for debugging
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
}
