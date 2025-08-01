package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	billing "ibp-geodns/src/IBPCollator/billing"
	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

type DowntimeEvent struct {
	ID         int64      `json:"id"`
	MemberName string     `json:"member_name"`
	CheckType  string     `json:"check_type"`
	CheckName  string     `json:"check_name"`
	DomainName string     `json:"domain_name,omitempty"`
	Endpoint   string     `json:"endpoint,omitempty"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty"`
	Duration   string     `json:"duration,omitempty"`
	Error      string     `json:"error,omitempty"`
	IsIPv6     bool       `json:"is_ipv6"`
	Status     string     `json:"status"` // "ongoing" or "resolved"
}

func handleDowntimeEvents(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Get and validate filters
	member := sanitizeString(r.URL.Query().Get("member"))
	service := sanitizeString(r.URL.Query().Get("service"))
	domain := sanitizeString(r.URL.Query().Get("domain"))
	checkType := sanitizeString(r.URL.Query().Get("check_type"))
	status := sanitizeString(r.URL.Query().Get("status"))

	// Validate member name (use new function that allows spaces)
	if member != "" && !validateMemberName(member) {
		writeError(w, http.StatusBadRequest, "Invalid member name")
		return
	}

	// Validate service name
	if service != "" && !validateIdentifier(service) {
		writeError(w, http.StatusBadRequest, "Invalid service name")
		return
	}

	// Validate domain
	if domain != "" && !validateIdentifier(domain) {
		writeError(w, http.StatusBadRequest, "Invalid domain")
		return
	}

	// Validate check type
	if checkType != "" && checkType != "site" && checkType != "domain" && checkType != "endpoint" {
		writeError(w, http.StatusBadRequest, "Invalid check type")
		return
	}

	// Validate status
	if status != "" && status != "ongoing" && status != "resolved" {
		writeError(w, http.StatusBadRequest, "Invalid status")
		return
	}

	query := `
		SELECT 
			id,
			member_name,
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			end_time,
			COALESCE(error, '') as error,
			is_ipv6
		FROM member_events
		WHERE status = 0
		AND start_time <= ?
		AND (end_time IS NULL OR end_time >= ?)
	`

	args := []interface{}{end, start}

	// Apply filters with parameterized queries
	if member != "" {
		query += " AND member_name = ?"
		args = append(args, member)
	}

	if domain != "" {
		query += " AND domain_name = ?"
		args = append(args, domain)
	}

	if checkType != "" {
		query += " AND check_type = ?"
		args = append(args, checkType)
	}

	if status == "ongoing" {
		query += " AND end_time IS NULL"
	} else if status == "resolved" {
		query += " AND end_time IS NOT NULL"
	}

	query += " ORDER BY start_time DESC"

	rows, err := data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query downtime events: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var events []DowntimeEvent
	for rows.Next() {
		var event DowntimeEvent
		var endTime sql.NullTime
		var domainName, endpoint, errorText sql.NullString
		var isIPv6 int

		err := rows.Scan(
			&event.ID,
			&event.MemberName,
			&event.CheckType,
			&event.CheckName,
			&domainName,
			&endpoint,
			&event.StartTime,
			&endTime,
			&errorText,
			&isIPv6,
		)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan downtime event: %v", err)
			continue
		}

		event.IsIPv6 = isIPv6 == 1

		if domainName.Valid {
			event.DomainName = domainName.String
		}
		if endpoint.Valid {
			event.Endpoint = endpoint.String
		}
		if errorText.Valid {
			event.Error = errorText.String
		}

		if endTime.Valid {
			event.EndTime = &endTime.Time
			event.Status = "resolved"
			duration := endTime.Time.Sub(event.StartTime)
			event.Duration = formatDuration(duration)
		} else {
			event.Status = "ongoing"
			duration := time.Now().UTC().Sub(event.StartTime)
			event.Duration = formatDuration(duration)
		}

		// Filter by service if specified
		if service != "" && event.DomainName != "" {
			serviceName := domainToServiceName(event.DomainName)
			if serviceName != service {
				continue
			}
		}

		events = append(events, event)
	}

	writeJSON(w, http.StatusOK, events)
}

func handleCurrentDowntime(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT 
			id,
			member_name,
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			COALESCE(error, '') as error,
			is_ipv6
		FROM member_events
		WHERE status = 0
		AND end_time IS NULL
		ORDER BY start_time DESC
	`

	rows, err := data2.DB.Query(query)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query current downtime: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var events []DowntimeEvent
	for rows.Next() {
		var event DowntimeEvent
		var domainName, endpoint, errorText sql.NullString
		var isIPv6 int

		err := rows.Scan(
			&event.ID,
			&event.MemberName,
			&event.CheckType,
			&event.CheckName,
			&domainName,
			&endpoint,
			&event.StartTime,
			&errorText,
			&isIPv6,
		)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan current downtime: %v", err)
			continue
		}

		event.IsIPv6 = isIPv6 == 1
		event.Status = "ongoing"

		if domainName.Valid {
			event.DomainName = domainName.String
		}
		if endpoint.Valid {
			event.Endpoint = endpoint.String
		}
		if errorText.Valid {
			event.Error = errorText.String
		}

		duration := time.Now().UTC().Sub(event.StartTime)
		event.Duration = formatDuration(duration)

		events = append(events, event)
	}

	writeJSON(w, http.StatusOK, events)
}

func handleDowntimeSummary(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Get total downtime events
	var totalEvents, ongoingEvents, resolvedEvents int
	var totalDowntimeMinutes float64

	// Total events
	err = data2.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM member_events 
		WHERE status = 0
		AND start_time <= ?
		AND (end_time IS NULL OR end_time >= ?)
	`, end, start).Scan(&totalEvents)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to get total events: %v", err)
	}

	// Ongoing events
	err = data2.DB.QueryRow(`
		SELECT COUNT(*) 
		FROM member_events 
		WHERE status = 0
		AND start_time <= ?
		AND end_time IS NULL
	`, end).Scan(&ongoingEvents)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to get ongoing events: %v", err)
	}

	resolvedEvents = totalEvents - ongoingEvents

	// Calculate total downtime
	rows, err := data2.DB.Query(`
		SELECT 
			start_time,
			COALESCE(end_time, ?) as end_time
		FROM member_events
		WHERE status = 0
		AND start_time <= ?
		AND (end_time IS NULL OR end_time >= ?)
	`, time.Now().UTC(), end, start)

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var eventStart, eventEnd time.Time
			rows.Scan(&eventStart, &eventEnd)

			// Adjust to date range
			if eventStart.Before(start) {
				eventStart = start
			}
			if eventEnd.After(end) {
				eventEnd = end
			}

			totalDowntimeMinutes += eventEnd.Sub(eventStart).Minutes()
		}
	}

	// Get affected members count
	var affectedMembers int
	data2.DB.QueryRow(`
		SELECT COUNT(DISTINCT member_name) 
		FROM member_events 
		WHERE status = 0
		AND start_time <= ?
		AND (end_time IS NULL OR end_time >= ?)
	`, end, start).Scan(&affectedMembers)

	summary := map[string]interface{}{
		"start_date":           start.Format("2006-01-02"),
		"end_date":             end.Format("2006-01-02"),
		"total_events":         totalEvents,
		"ongoing_events":       ongoingEvents,
		"resolved_events":      resolvedEvents,
		"affected_members":     affectedMembers,
		"total_downtime_hours": totalDowntimeMinutes / 60,
		"average_downtime_hours": func() float64 {
			if resolvedEvents > 0 {
				return (totalDowntimeMinutes / 60) / float64(resolvedEvents)
			}
			return 0
		}(),
	}

	writeJSON(w, http.StatusOK, summary)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

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
func CalculateSLAAdjustments(month time.Time, sum *billing.Summary) (SLASummary, error) {
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

	// Get configuration for member name mapping
	c := cfg.GetConfig()

	// Query ALL downtime events (both closed and open)
	query := `
		SELECT 
			member_name,
			domain_name,
			check_type,
			start_time,
			CASE 
				WHEN end_time IS NULL THEN NOW()
				ELSE end_time 
			END as calc_end_time,
			is_ipv6
		FROM member_events
		WHERE status = 0
		AND start_time < ?
		AND (end_time IS NULL OR end_time > ?)
		ORDER BY member_name, start_time
	`

	rows, err := data2.DB.Query(query, endTime, startTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query downtime events: %w", err)
	}
	defer rows.Close()

	// Track downtime by member and service
	// Map from memberID -> serviceName -> accumulated downtime hours
	memberServiceDowntime := make(map[string]map[string]float64)
	// Track site-level downtime separately
	memberSiteDowntime := make(map[string]float64)

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

		// Map database member name to config member ID
		memberID := ""
		for id, member := range c.Members {
			if member.Details.Name == memberName {
				memberID = id
				break
			}
		}
		if memberID == "" {
			// Fallback to using the name as-is
			memberID = memberName
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

		// Handle site-level checks (affects ALL services)
		if checkType == "site" {
			memberSiteDowntime[memberID] += downtimeHours
			log.Log(log.Debug, "[SLA] Site downtime for %s: +%.2f hours (total: %.2f)",
				memberID, downtimeHours, memberSiteDowntime[memberID])
		} else {
			// Map domain to service for domain/endpoint checks
			serviceName := mapDomainToService(domainName.String, checkType)
			if serviceName == "" {
				continue
			}

			// Initialize maps if needed
			if _, exists := memberServiceDowntime[memberID]; !exists {
				memberServiceDowntime[memberID] = make(map[string]float64)
			}

			// Add to service-specific downtime
			memberServiceDowntime[memberID][serviceName] += downtimeHours
			log.Log(log.Debug, "[SLA] Service downtime for %s/%s: +%.2f hours (total: %.2f)",
				memberID, serviceName, downtimeHours, memberServiceDowntime[memberID][serviceName])
		}
	}

	// Build the SLA summary
	for memberID, m := range sum.Members {
		if _, ok := out[memberID]; !ok {
			out[memberID] = make(map[string]SLABreakdown)
		}

		// Get site-level downtime for this member
		siteDowntime := memberSiteDowntime[memberID]

		for svcKey := range m.ServiceCosts {
			// Start with site-level downtime (affects all services)
			downtime := siteDowntime

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

			if downtime > 0 {
				log.Log(log.Debug, "[SLA] %s/%s - Total downtime: %.2f hours (%.2f%% uptime)",
					memberID, svcKey, downtime, uptimePercent)
			}
		}
	}

	return out, nil
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
