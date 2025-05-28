package IBPCollator

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	natsconn "ibp-geodns/src/common/nats"
)

// CollatorMain is the entry point. It loads config, connects to NATS, and handles user commands.
func CollatorMain() {
	configPath := flag.String("config", "config.json", "Path to configuration file for the Collator.")
	action := flag.String("action", "", "Which action to perform: 'usage' or 'downtime'.")
	startArg := flag.String("start", "", "Start date/time (for usage or downtime). e.g. '2024-01-01' or '2024-01-01T00:00:00Z'")
	endArg := flag.String("end", "", "End date/time (for usage or downtime). e.g. '2024-01-31' or '2024-01-31T23:59:59Z'")
	domainArg := flag.String("domain", "", "Domain filter for usage queries.")
	memberArg := flag.String("member", "", "Member filter for usage/downtime queries.")
	countryArg := flag.String("country", "", "Country filter for usage queries.")
	timeoutArg := flag.Int("timeoutSec", 5, "Seconds to wait for responses from each request.")
	flag.Parse()

	if *action == "" {
		fmt.Println("No action specified. Use -action=usage or -action=downtime.")
		os.Exit(1)
	}

	// Load config
	cfg.Init(*configPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// Connect to NATS
	err := natsconn.Connect()
	if err != nil {
		log.Log(log.Fatal, "Failed to connect to NATS: %v", err)
		os.Exit(1)
	}
	defer natsconn.Disconnect()

	// Enable Collator role. This sets up any needed NATS subscriptions or state.
	err = natsconn.EnableCollatorRole()
	if err != nil {
		log.Log(log.Fatal, "Failed to enable Collator role: %v", err)
		os.Exit(1)
	}

	switch strings.ToLower(*action) {
	case "usage":
		if *startArg == "" || *endArg == "" {
			fmt.Println("For usage, you must specify -start and -end.")
			os.Exit(1)
		}
		usageOutput, err := RequestUsageFromAll(*startArg, *endArg, *domainArg, *memberArg, *countryArg, time.Duration(*timeoutArg)*time.Second)
		if err != nil {
			log.Log(log.Error, "Usage request error: %v", err)
			os.Exit(1)
		}
		// Print or process usageOutput
		fmt.Println("Aggregated usage records:")
		for _, rec := range usageOutput {
			fmt.Printf("Date=%s, Domain=%s, Member=%s, Country=%s, Hits=%d\n",
				rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
		}

	case "downtime":
		if *startArg == "" || *endArg == "" {
			fmt.Println("For downtime, you must specify -start and -end.")
			os.Exit(1)
		}
		startT, endT, parseErr := parseTimeRange(*startArg, *endArg)
		if parseErr != nil {
			log.Log(log.Error, "Failed to parse time range: %v", parseErr)
			os.Exit(1)
		}
		downtimeEvents, err := RequestDowntimeFromAll(startT, endT, *memberArg, time.Duration(*timeoutArg)*time.Second)
		if err != nil {
			log.Log(log.Error, "Downtime request error: %v", err)
			os.Exit(1)
		}
		// Print or process downtimeEvents
		fmt.Println("Aggregated downtime events:")
		for _, evt := range downtimeEvents {
			fmt.Printf("Member=%s, CheckType=%s, Start=%s, End=%s, Status=%v\n",
				evt.MemberName, evt.CheckType, evt.StartTime.Format(time.RFC3339), evt.EndTime.Format(time.RFC3339), evt.Status)
		}

	default:
		fmt.Printf("Unknown action: %s\n", *action)
		os.Exit(1)
	}
}

// RequestUsageFromAll sends a usage request to "dns.usage.getUsage" and aggregates all replies.
func RequestUsageFromAll(startDate, endDate, domain, member, country string, timeout time.Duration) ([]natsconn.UsageRecord, error) {
	req := natsconn.UsageRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Domain:     domain,
		MemberName: member,
		Country:    country,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal usage request: %w", err)
	}

	// Create an Inbox for direct replies
	inbox := natsconn.State.NodeID + ".dns.usage.reply." + generateIDSuffix()
	sub, subErr := natsconn.Subscribe(inbox, func(m *natsconn.NatsMsg) {
		// We'll handle these in a separate goroutine or aggregator
	})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}
	defer sub.Unsubscribe()

	// We gather responses in an aggregator
	var mu sync.Mutex
	var aggregated []natsconn.UsageRecord

	sub.SetPendingLimits(-1, -1) // allow large messages
	sub.Callback = func(m *natsconn.NatsMsg) {
		var resp natsconn.UsageResponse
		if unErr := json.Unmarshal(m.Data, &resp); unErr == nil {
			mu.Lock()
			aggregated = append(aggregated, resp.UsageRecords...)
			mu.Unlock()
		}
	}

	// Publish a request to "dns.usage.getUsage", specifying the Inbox as the reply subject
	err = natsconn.PublishRequest("dns.usage.getUsage", inbox, data)
	if err != nil {
		return nil, fmt.Errorf("publish usage request error: %w", err)
	}

	// Wait up to 'timeout' for responses
	time.Sleep(timeout)

	return aggregated, nil
}

// RequestDowntimeFromAll sends a downtime request to "monitor.stats.getDowntime" and aggregates all replies.
func RequestDowntimeFromAll(startTime, endTime time.Time, memberName string, timeout time.Duration) ([]natsconn.DowntimeEvent, error) {
	req := natsconn.DowntimeRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		MemberName: memberName,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal downtime request: %w", err)
	}

	inbox := natsconn.State.NodeID + ".monitor.stats.reply." + generateIDSuffix()
	sub, subErr := natsconn.Subscribe(inbox, func(m *natsconn.NatsMsg) {})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}
	defer sub.Unsubscribe()

	var mu sync.Mutex
	var aggregated []natsconn.DowntimeEvent

	sub.SetPendingLimits(-1, -1)
	sub.Callback = func(m *natsconn.NatsMsg) {
		var resp natsconn.DowntimeResponse
		if unErr := json.Unmarshal(m.Data, &resp); unErr == nil {
			mu.Lock()
			aggregated = append(aggregated, resp.Events...)
			mu.Unlock()
		}
	}

	err = natsconn.PublishRequest("monitor.stats.getDowntime", inbox, data)
	if err != nil {
		return nil, fmt.Errorf("publish downtime request error: %w", err)
	}

	time.Sleep(timeout)

	return aggregated, nil
}

// parseTimeRange tries to parse start/end as RFC3339 first, else as date.
func parseTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	var startT, endT time.Time
	var err error
	// Attempt RFC3339
	startT, err = time.Parse(time.RFC3339, startStr)
	if err != nil {
		// Maybe parse as date only
		startT, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("cannot parse start time: %v", err)
		}
	}
	endT, err = time.Parse(time.RFC3339, endStr)
	if err != nil {
		// Maybe parse as date only
		endT, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("cannot parse end time: %v", err)
		}
	}
	return startT, endT, nil
}

// generateIDSuffix creates a short suffix for an Inbox subject.
func generateIDSuffix() string {
	// This can be replaced with a random string or UUID.
	// For brevity, a timestamp-based string:
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// main is a wrapper for the CollatorMain if you want a direct 'go run' approach.
func main() {
	CollatorMain()
}
