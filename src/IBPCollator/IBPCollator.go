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
	// Parse flags
	configPath := flag.String("config", "config.json", "Path to config file")
	flag.Parse()

	// Load config
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

	// Enable Collator role
	err = nconn.EnableCollatorRole()
	if err != nil {
		log.Log(log.Fatal, "[Collator] Failed to enable Collator role: %v", err)
		os.Exit(1)
	}

	// We'll create usage and downtime requests, then gather data from the cluster.

	// 1) Usage request (all dnsApi)
	log.Log(log.Info, "[Collator] Requesting usage data from all IBPDns nodes...")
	usageReq := nconn.UsageRequest{
		StartDate:  "2025-05-30",
		EndDate:    "2025-05-30",
		Domain:     "", // all domains
		MemberName: "", // all members
		Country:    "", // all countries
	}
	usageTimeout := 5 * time.Second

	usageRecords, err := nconn.RequestAllDnsUsage(usageReq, usageTimeout)
	if err != nil {
		log.Log(log.Error, "[Collator] Usage request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d usage records", len(usageRecords))
	}

	// 2) Downtime request (all monitor)
	log.Log(log.Info, "[Collator] Requesting downtime data from all IBPMonitor nodes...")
	dtReq := nconn.DowntimeRequest{
		StartTime:  time.Date(2025, 05, 30, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2025, 05, 30, 23, 59, 59, 0, time.UTC),
		MemberName: "", // all members
	}
	dtTimeout := 5 * time.Second

	dtEvents, err := nconn.RequestAllMonitorsDowntime(dtReq, dtTimeout)
	if err != nil {
		log.Log(log.Error, "[Collator] Downtime request error: %v", err)
	} else {
		log.Log(log.Info, "[Collator] Received total of %d downtime events", len(dtEvents))
	}

	// Print usage + downtime to console (for now). Then exit.
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
