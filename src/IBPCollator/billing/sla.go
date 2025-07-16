package billing

import (
	"database/sql"
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

	// Total hours in the month
	totalHours := endTime.Sub(startTime).Hours()
	slaHours := totalHours * (DefaultSLAPercentage / 100.0)

	// Query downtime events for all members
	query := `
		SELECT 
			member_name,
			domain_name,
			check_type,
			start_time,
			COALESCE(end_time, ?) as end_time,
			is_ipv6
		FROM member_events
		WHERE status = 0
		AND start_time < ?
		AND (end_time IS NULL OR end_time > ?)
		ORDER BY member_name, domain_name, start_time
	`

	rows, err := data2.DB.Query(query, endTime, endTime, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query downtime events: %w", err)
	}
	defer rows.Close()

	// Track downtime by member and service
	memberServiceDowntime := make(map[string]map[string]float64)

	for rows.Next() {
		var memberName, checkType string
		var domainName sql.NullString
		var startTimeRaw, endTimeRaw time.Time
		var isIPv6 bool

		err := rows.Scan(&memberName, &domainName, &checkType, &startTimeRaw, &endTimeRaw, &isIPv6)
		if err != nil {
			log.Log(log.Error, "Failed to scan downtime row: %v", err)
			continue
		}

		// Adjust times to be within the month
		eventStart := startTimeRaw
		if eventStart.Before(startTime) {
			eventStart = startTime
		}

		eventEnd := endTimeRaw
		if eventEnd.After(endTime) {
			eventEnd = endTime
		}

		// Calculate downtime hours for this event
		downtimeHours := eventEnd.Sub(eventStart).Hours()

		// Map domain to service
		serviceName := mapDomainToService(domainName.String, checkType)
		if serviceName == "" {
			continue
		}

		// Initialize maps if needed
		if _, exists := memberServiceDowntime[memberName]; !exists {
			memberServiceDowntime[memberName] = make(map[string]float64)
		}

		// Add to downtime
		memberServiceDowntime[memberName][serviceName] += downtimeHours
	}

	// Build the SLA summary
	for memberID, m := range sum.Members {
		if _, ok := out[memberID]; !ok {
			out[memberID] = make(map[string]SLABreakdown)
		}

		// Check if there's a site-level downtime
		siteDowntime := 0.0
		if allServicesDowntime, exists := memberServiceDowntime[memberID]; exists {
			if siteDown, exists2 := allServicesDowntime["ALL_SERVICES"]; exists2 {
				siteDowntime = siteDown
			}
		}

		for svcKey := range m.ServiceCosts {
			downtime := siteDowntime // Start with site-level downtime

			// Add service-specific downtime
			if memberDowntime, exists := memberServiceDowntime[memberID]; exists {
				if svcDowntime, exists2 := memberDowntime[svcKey]; exists2 {
					downtime += svcDowntime
				}
			}

			// Cap downtime at total hours
			if downtime > totalHours {
				downtime = totalHours
			}

			uptime := totalHours - downtime
			uptimePercent := (uptime / totalHours) * 100.0
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
		}
	}

	return out, nil
}

// mapDomainToService maps a domain name to a service name
func mapDomainToService(domain, checkType string) string {
	if checkType == "site" {
		// Site-level checks affect all services
		return "ALL_SERVICES"
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
