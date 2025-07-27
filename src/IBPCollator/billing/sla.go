package billing

import (
	"fmt"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

// SLABreakdown captures the availability of a single <member,service> pair.
type SLABreakdown struct {
	HoursTotal   float64
	HoursDown    float64
	HoursUp      float64
	Uptime       float64 // 0-100 percentage
	SLAThreshold float64 // SLA threshold in percentage (e.g., 99.99)
	SLAHours     float64 // SLA threshold in hours
	MeetsSLA     bool
}

// SLASummary maps member → service → breakdown.
type SLASummary map[string]map[string]SLABreakdown

// Default SLA threshold
const DefaultSLAPercentage = 99.99

// downtimePeriod represents a period of downtime
type downtimePeriod struct {
	start time.Time
	end   time.Time
}

// CalculateSLAAdjustments calculates actual uptime from the member_events table
func CalculateSLAAdjustments(month time.Time, sum *Summary) (SLASummary, error) {
	out := make(SLASummary)

	// Check if database is initialized
	if data2.DB == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	// Calculate the time range for the month
	startTime := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 1, 0).Add(-time.Second)
	now := time.Now().UTC()

	// If calculating for current or future month, use current time as end
	if endTime.After(now) {
		endTime = now
	}

	// Total hours in the month (up to current time if applicable)
	totalHours := endTime.Sub(startTime).Hours()
	slaHours := totalHours * (DefaultSLAPercentage / 100.0)

	// Get configuration for member name mapping
	c := cfg.GetConfig()

	// Build member ID to DB name mapping
	memberIDToDBName := make(map[string]string)
	for id, member := range c.Members {
		if member.Details.Name != "" {
			memberIDToDBName[id] = member.Details.Name
		} else {
			memberIDToDBName[id] = id
		}
	}

	// Build service to domains mapping
	serviceToDomains := make(map[string][]string)
	for svcName, svc := range c.Services {
		domains := []string{}
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if domain := extractDomainFromURL(rpcUrl); domain != "" {
					domains = append(domains, domain)
				}
			}
		}
		serviceToDomains[svcName] = domains
	}

	// Calculate downtime for each member/service combination
	for memberID, m := range sum.Members {
		if _, ok := out[memberID]; !ok {
			out[memberID] = make(map[string]SLABreakdown)
		}

		dbMemberName := memberIDToDBName[memberID]

		for svcKey := range m.ServiceCosts {
			// Calculate downtime for this specific service
			downtime := calculateServiceDowntime(dbMemberName, svcKey, serviceToDomains[svcKey], startTime, endTime)

			// Calculate uptime
			uptime := totalHours - downtime
			if uptime < 0 {
				uptime = 0
			}

			uptimePercent := 100.0
			if totalHours > 0 {
				uptimePercent = (uptime / totalHours) * 100.0
			}

			meetsSLA := uptimePercent >= DefaultSLAPercentage

			out[memberID][svcKey] = SLABreakdown{
				HoursTotal:   totalHours,
				HoursDown:    downtime,
				HoursUp:      uptime,
				Uptime:       uptimePercent,
				SLAThreshold: DefaultSLAPercentage,
				SLAHours:     slaHours,
				MeetsSLA:     meetsSLA,
			}

			if downtime > 0 {
				log.Log(log.Debug, "[SLA] %s/%s - Total downtime: %.2f hours (%.2f%% uptime)",
					memberID, svcKey, downtime, uptimePercent)
			}
		}
	}

	return out, nil
}

// calculateServiceDowntime calculates total downtime hours for a specific service
func calculateServiceDowntime(memberName, serviceName string, domains []string, startTime, endTime time.Time) float64 {
	if data2.DB == nil {
		return 0
	}

	totalDowntime := 0.0

	// Query for site-level checks (affects all services)
	siteQuery := `
		SELECT 
			start_time,
			CASE 
				WHEN end_time IS NULL THEN ?
				ELSE end_time 
			END as calc_end_time
		FROM member_events
		WHERE member_name = ?
		AND check_type = 'site'
		AND status = 0
		AND start_time < ?
		AND (end_time IS NULL OR end_time > ?)
	`

	rows, err := data2.DB.Query(siteQuery, endTime, memberName, endTime, startTime)
	if err != nil {
		log.Log(log.Error, "[SLA] Failed to query site downtime: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var eventStart, eventEnd time.Time
			if err := rows.Scan(&eventStart, &eventEnd); err != nil {
				log.Log(log.Error, "[SLA] Failed to scan site event: %v", err)
				continue
			}

			// Adjust to month boundaries
			if eventStart.Before(startTime) {
				eventStart = startTime
			}
			if eventEnd.After(endTime) {
				eventEnd = endTime
			}

			downtime := eventEnd.Sub(eventStart).Hours()
			totalDowntime += downtime
			log.Log(log.Debug, "[SLA] Site downtime for %s: %.2f hours", memberName, downtime)
		}
	}

	// Query for domain/endpoint checks specific to this service
	if len(domains) > 0 {
		// Build parameterized query
		placeholders := make([]string, len(domains))
		args := make([]interface{}, 0, len(domains)+4)
		args = append(args, endTime, memberName)

		for i, domain := range domains {
			placeholders[i] = "?"
			args = append(args, domain)
		}

		args = append(args, endTime, startTime)

		serviceQuery := fmt.Sprintf(`
			SELECT 
				check_type,
				domain_name,
				endpoint,
				start_time,
				CASE 
					WHEN end_time IS NULL THEN ?
					ELSE end_time 
				END as calc_end_time
			FROM member_events
			WHERE member_name = ?
			AND check_type IN ('domain', 'endpoint')
			AND status = 0
			AND domain_name IN (%s)
			AND start_time < ?
			AND (end_time IS NULL OR end_time > ?)
		`, strings.Join(placeholders, ","))

		rows, err := data2.DB.Query(serviceQuery, args...)
		if err != nil {
			log.Log(log.Error, "[SLA] Failed to query service downtime: %v", err)
		} else {
			defer rows.Close()

			// Track unique downtime periods to avoid double-counting overlaps
			periods := []downtimePeriod{}

			for rows.Next() {
				var checkType, domainName, endpoint string
				var eventStart, eventEnd time.Time

				if err := rows.Scan(&checkType, &domainName, &endpoint, &eventStart, &eventEnd); err != nil {
					log.Log(log.Error, "[SLA] Failed to scan service event: %v", err)
					continue
				}

				// Adjust to month boundaries
				if eventStart.Before(startTime) {
					eventStart = startTime
				}
				if eventEnd.After(endTime) {
					eventEnd = endTime
				}

				periods = append(periods, downtimePeriod{start: eventStart, end: eventEnd})

				log.Log(log.Debug, "[SLA] %s downtime for %s/%s (domain: %s): %.2f hours",
					checkType, memberName, serviceName, domainName, eventEnd.Sub(eventStart).Hours())
			}

			// Merge overlapping periods
			if len(periods) > 0 {
				merged := mergeOverlappingPeriods(periods)
				for _, period := range merged {
					totalDowntime += period.end.Sub(period.start).Hours()
				}
			}
		}
	}

	return totalDowntime
}

// mergeOverlappingPeriods merges overlapping downtime periods to avoid double-counting
func mergeOverlappingPeriods(periods []downtimePeriod) []downtimePeriod {
	if len(periods) <= 1 {
		return periods
	}

	// Sort by start time
	for i := 0; i < len(periods)-1; i++ {
		for j := i + 1; j < len(periods); j++ {
			if periods[j].start.Before(periods[i].start) {
				periods[i], periods[j] = periods[j], periods[i]
			}
		}
	}

	merged := []downtimePeriod{periods[0]}
	for i := 1; i < len(periods); i++ {
		last := &merged[len(merged)-1]
		current := periods[i]

		// If current period overlaps with last, merge them
		if current.start.Before(last.end) || current.start.Equal(last.end) {
			if current.end.After(last.end) {
				last.end = current.end
			}
		} else {
			// No overlap, add as new period
			merged = append(merged, current)
		}
	}

	return merged
}

// extractDomainFromURL extracts the domain from an RPC URL
func extractDomainFromURL(rpcUrl string) string {
	// Remove protocol
	url := strings.TrimPrefix(rpcUrl, "wss://")
	url = strings.TrimPrefix(url, "ws://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Remove path and port
	if idx := strings.Index(url, "/"); idx != -1 {
		url = url[:idx]
	}
	if idx := strings.Index(url, ":"); idx != -1 {
		url = url[:idx]
	}

	return strings.ToLower(url)
}

// mapDomainToService maps a domain name to a service name
func mapDomainToService(domain, checkType string) string {
	if checkType == "site" {
		// Site-level checks don't map to a specific service
		return ""
	}

	if domain == "" {
		return ""
	}

	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				// Clean up the URL for comparison
				cleanUrl := strings.ToLower(strings.TrimSpace(rpcUrl))
				cleanDomain := strings.ToLower(strings.TrimSpace(domain))

				// Check if the domain is contained in the RPC URL
				if strings.Contains(cleanUrl, cleanDomain) {
					return svcName
				}

				// Also check if the RPC URL contains the domain without protocol
				if strings.Contains(cleanUrl, "://"+cleanDomain) ||
					strings.Contains(cleanUrl, "://"+cleanDomain+":") ||
					strings.Contains(cleanUrl, "://"+cleanDomain+"/") {
					return svcName
				}
			}
		}
	}

	// If no match found, log it for debugging
	log.Log(log.Debug, "[SLA] Could not map domain '%s' to any service", domain)
	return ""
}
