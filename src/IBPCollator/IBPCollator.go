package IBPCollator

import (
	"flag"
	"fmt"
	"os"
	"time"

	"ibp-geodns/src/IBPCollator/nats/downtime"
	"ibp-geodns/src/IBPCollator/nats/usage"
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	nconn "ibp-geodns/src/common/nats"
)

// CollatorMain can be the main entry point for the IBPCollator process.
func CollatorMain() {
	configPath := flag.String("config", "config.json", "Path to config file.")
	action := flag.String("action", "", "Action: usage or downtime.")
	startArg := flag.String("start", "", "Start date/time or date only.")
	endArg := flag.String("end", "", "End date/time or date only.")
	domainArg := flag.String("domain", "", "Domain filter for usage.")
	memberArg := flag.String("member", "", "Member filter for usage/downtime.")
	countryArg := flag.String("country", "", "Country filter for usage.")
	timeoutSec := flag.Int("timeoutSec", 5, "Timeout in seconds for replies.")
	flag.Parse()

	if *action == "" {
		fmt.Println("ERROR: No action specified. Use -action=usage or -action=downtime.")
		os.Exit(1)
	}

	// Initialize config
	cfg.Init(*configPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// Connect to NATS
	err := nconn.Connect()
	if err != nil {
		log.Log(log.Fatal, "Collator: failed to connect to NATS: %v", err)
		os.Exit(1)
	}
	defer nconn.Disconnect()

	// Enable Collator role in the common NATS package
	err = nconn.EnableCollatorRole()
	if err != nil {
		log.Log(log.Fatal, "Collator: failed to enable collator role: %v", err)
		os.Exit(1)
	}

	switch *action {
	case "usage":
		if *startArg == "" || *endArg == "" {
			fmt.Println("ERROR: For usage, please specify -start and -end.")
			os.Exit(1)
		}
		aggUsage, err := usage.RequestAllUsage(*startArg, *endArg, *domainArg, *memberArg, *countryArg, time.Duration(*timeoutSec)*time.Second)
		if err != nil {
			log.Log(log.Error, "Collator: usage request error: %v", err)
			os.Exit(1)
		}
		fmt.Println("=== Aggregated Usage Records ===")
		for _, rec := range aggUsage {
			fmt.Printf("Date=%s Domain=%s Member=%s Country=%s Hits=%d\n",
				rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
		}

	case "downtime":
		if *startArg == "" || *endArg == "" {
			fmt.Println("ERROR: For downtime, please specify -start and -end.")
			os.Exit(1)
		}
		startT, endT, parseErr := downtime.ParseTimeRange(*startArg, *endArg)
		if parseErr != nil {
			log.Log(log.Error, "Collator: cannot parse time range: %v", parseErr)
			os.Exit(1)
		}
		events, err := downtime.RequestAllDowntime(startT, endT, *memberArg, time.Duration(*timeoutSec)*time.Second)
		if err != nil {
			log.Log(log.Error, "Collator: downtime request error: %v", err)
			os.Exit(1)
		}
		fmt.Println("=== Aggregated Downtime Events ===")
		for _, evt := range events {
			fmt.Printf("Member=%s CheckType=%s Start=%s End=%s Status=%v\n",
				evt.MemberName, evt.CheckType,
				evt.StartTime.Format(time.RFC3339),
				evt.EndTime.Format(time.RFC3339),
				evt.Status)
		}
	default:
		fmt.Printf("ERROR: Unknown action '%s'\n", *action)
		os.Exit(1)
	}
}

// main allows a direct `go run collator.go`.
func main() {
	CollatorMain()
}
