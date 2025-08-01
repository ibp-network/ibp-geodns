package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	billing "ibp-geodns/src/IBPCollator/billing"
	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

type BillingMember struct {
	Name         string           `json:"name"`
	Level        int              `json:"level"`
	Services     []BillingService `json:"services"`
	TotalBase    float64          `json:"total_base_cost"`
	TotalBilled  float64          `json:"total_billed"`
	TotalCredits float64          `json:"total_credits"`
	MeetsSLA     bool             `json:"meets_sla"`
}

type BillingService struct {
	Name       string          `json:"name"`
	BaseCost   float64         `json:"base_cost"`
	Uptime     float64         `json:"uptime_percentage"`
	BilledCost float64         `json:"billed_cost"`
	Credits    float64         `json:"credits"`
	MeetsSLA   bool            `json:"meets_sla"`
	Downtime   []DowntimeEvent `json:"downtime_events,omitempty"`
}

func handleBillingBreakdown(w http.ResponseWriter, r *http.Request) {
	// Parse month and year
	monthStr := r.URL.Query().Get("month")
	yearStr := r.URL.Query().Get("year")

	if monthStr == "" || yearStr == "" {
		// Default to previous month
		now := time.Now().UTC()
		prevMonth := now.AddDate(0, -1, 0)
		monthStr = strconv.Itoa(int(prevMonth.Month()))
		yearStr = strconv.Itoa(prevMonth.Year())
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		writeError(w, http.StatusBadRequest, "Invalid month")
		return
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2020 || year > 2100 {
		writeError(w, http.StatusBadRequest, "Invalid year")
		return
	}

	billingMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)

	// Get current billing summary
	summary := billing.GetSummary()

	// Calculate SLA for the month
	sla, err := billing.CalculateSLAAdjustments(billingMonth, &summary)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to calculate SLA: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to calculate SLA")
		return
	}

	// Get member filter if specified
	memberFilter := r.URL.Query().Get("member")

	var billingMembers []BillingMember

	for memberName, memberCost := range summary.Members {
		if memberFilter != "" && memberFilter != memberName {
			continue
		}

		c := cfg.GetConfig()
		memberConfig, exists := c.Members[memberName]
		if !exists {
			continue
		}

		billingMember := BillingMember{
			Name:     memberName,
			Level:    memberConfig.Membership.Level,
			Services: []BillingService{},
		}

		memberMeetsSLA := true

		for serviceName, baseCost := range memberCost.ServiceCosts {
			breakdown := getSLABreakdown(sla, memberName, serviceName)
			billedCost := baseCost * (breakdown.Uptime / 100.0)
			credits := baseCost - billedCost

			service := BillingService{
				Name:       serviceName,
				BaseCost:   baseCost,
				Uptime:     breakdown.Uptime,
				BilledCost: billedCost,
				Credits:    credits,
				MeetsSLA:   breakdown.MeetsSLA,
			}

			if !breakdown.MeetsSLA {
				memberMeetsSLA = false
			}

			// Get downtime events for this service if requested
			if r.URL.Query().Get("include_downtime") == "true" {
				// Use the member's Details.Name for database lookup
				dbMemberName := memberName
				if memberConfig.Details.Name != "" {
					dbMemberName = memberConfig.Details.Name
				}
				service.Downtime = getServiceDowntimeForAPI(dbMemberName, serviceName, billingMonth)
			}

			billingMember.Services = append(billingMember.Services, service)
			billingMember.TotalBase += baseCost
			billingMember.TotalBilled += billedCost
			billingMember.TotalCredits += credits
		}

		billingMember.MeetsSLA = memberMeetsSLA
		billingMembers = append(billingMembers, billingMember)
	}

	result := map[string]interface{}{
		"month":   billingMonth.Format("2006-01"),
		"members": billingMembers,
		"total_base_cost": func() float64 {
			var total float64
			for _, m := range billingMembers {
				total += m.TotalBase
			}
			return total
		}(),
		"total_billed": func() float64 {
			var total float64
			for _, m := range billingMembers {
				total += m.TotalBilled
			}
			return total
		}(),
		"total_credits": func() float64 {
			var total float64
			for _, m := range billingMembers {
				total += m.TotalCredits
			}
			return total
		}(),
	}

	writeJSON(w, http.StatusOK, result)
}

func handleBillingSummary(w http.ResponseWriter, r *http.Request) {
	// Get current billing summary
	summary := billing.GetSummary()

	// Calculate totals
	var totalBaseCost, totalMembers, totalServices float64
	serviceCount := make(map[string]int)

	for _, memberCost := range summary.Members {
		totalMembers++
		for serviceName, cost := range memberCost.ServiceCosts {
			totalBaseCost += cost
			serviceCount[serviceName]++
			totalServices++
		}
	}

	// Get SLA summary for current month
	now := time.Now().UTC()
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	sla, _ := billing.CalculateSLAAdjustments(currentMonth, &summary)

	var totalCredits float64
	var slaViolations int

	for memberName, memberServices := range sla {
		for serviceName, breakdown := range memberServices {
			if memberCost, exists := summary.Members[memberName]; exists {
				if baseCost, exists := memberCost.ServiceCosts[serviceName]; exists {
					credit := baseCost * (1 - breakdown.Uptime/100.0)
					totalCredits += credit
					if !breakdown.MeetsSLA {
						slaViolations++
					}
				}
			}
		}
	}

	result := map[string]interface{}{
		"last_refresh":                 summary.Refresh,
		"total_members":                int(totalMembers),
		"total_services":               int(totalServices),
		"unique_services":              len(serviceCount),
		"total_base_cost_monthly":      totalBaseCost,
		"current_month_credits":        totalCredits,
		"current_month_sla_violations": slaViolations,
		"service_distribution":         serviceCount,
	}

	writeJSON(w, http.StatusOK, result)
}

func getSLABreakdown(sla billing.SLASummary, member, service string) billing.SLABreakdown {
	if memberServices, ok := sla[member]; ok {
		if breakdown, ok := memberServices[service]; ok {
			return breakdown
		}
	}
	// Return default if not found
	return billing.SLABreakdown{
		HoursTotal:   730,
		HoursDown:    0,
		HoursUp:      730,
		Uptime:       100.0,
		SLAThreshold: billing.DefaultSLAPercentage,
		SLAHours:     730 * (billing.DefaultSLAPercentage / 100.0),
		MeetsSLA:     true,
	}
}

func getServiceDowntimeForAPI(memberName, serviceName string, month time.Time) []DowntimeEvent {
	events := []DowntimeEvent{}
	if data2.DB == nil {
		return events
	}

	// Map service to domains
	c := cfg.GetConfig()
	domains := []string{}
	if svc, exists := c.Services[serviceName]; exists {
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if domain := extractDomainFromURL(rpcUrl); domain != "" {
					domains = append(domains, domain)
				}
			}
		}
	}

	startTime := month
	endTime := month.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// First get site-level events that affect all services
	siteQuery := `
		SELECT  
			id,
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			end_time,
			COALESCE(error, '') as error,
			is_ipv6
		FROM member_events
		WHERE member_name = ?
		AND check_type = 'site'
		AND status = 0
		AND (
			(start_time < ? AND (end_time IS NULL OR end_time > ?))
			OR
			(start_time >= ? AND start_time < ?)
		)
		ORDER BY start_time DESC
	`

	rows, err := data2.DB.Query(siteQuery, memberName, endTime, startTime, startTime, endTime)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query site downtime events: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var event DowntimeEvent
			var domainName, endpoint, errorText sql.NullString
			var isIPv6 int
			var endTime sql.NullTime

			err := rows.Scan(
				&event.ID,
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
				log.Log(log.Error, "[CollatorAPI] Failed to scan site downtime event: %v", err)
				continue
			}

			event.MemberName = memberName
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

			// Adjust times to be within month
			if event.StartTime.Before(startTime) {
				event.StartTime = startTime
			}
			if endTime.Valid {
				event.EndTime = &endTime.Time
				if event.EndTime.After(month.AddDate(0, 1, 0).Add(-time.Nanosecond)) {
					t := month.AddDate(0, 1, 0).Add(-time.Nanosecond)
					event.EndTime = &t
				}
				event.Status = "resolved"
			} else {
				t := month.AddDate(0, 1, 0).Add(-time.Nanosecond)
				event.EndTime = &t
				event.Status = "ongoing"
			}

			// Calculate duration
			duration := event.EndTime.Sub(event.StartTime)
			event.Duration = formatDuration(duration)

			events = append(events, event)
		}
	}

	// Then get service-specific events
	if len(domains) == 0 {
		return events
	}

	// Build parameterized query with proper placeholders
	placeholders := make([]string, len(domains))
	args := make([]interface{}, 0, len(domains)+5)
	args = append(args, memberName)
	for i, domain := range domains {
		placeholders[i] = "?"
		args = append(args, domain)
	}
	args = append(args, endTime, startTime, startTime, endTime)

	// Safe query construction with parameterized inputs
	query := fmt.Sprintf(`
		SELECT  
			id,
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			end_time,
			COALESCE(error, '') as error,
			is_ipv6
		FROM member_events
		WHERE member_name = ?
		AND status = 0
		AND domain_name IN (%s)
		AND (
			(start_time < ? AND (end_time IS NULL OR end_time > ?))
			OR
			(start_time >= ? AND start_time < ?)
		)
		ORDER BY start_time DESC`, strings.Join(placeholders, ","))

	rows, err = data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query service downtime events: %v", err)
		return events
	}
	defer rows.Close()

	for rows.Next() {
		var event DowntimeEvent
		var domainName, endpoint, errorText sql.NullString
		var isIPv6 int
		var endTime sql.NullTime

		err := rows.Scan(
			&event.ID,
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

		event.MemberName = memberName
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

		// Adjust times to be within month
		if event.StartTime.Before(startTime) {
			event.StartTime = startTime
		}
		if endTime.Valid {
			event.EndTime = &endTime.Time
			if event.EndTime.After(month.AddDate(0, 1, 0).Add(-time.Nanosecond)) {
				t := month.AddDate(0, 1, 0).Add(-time.Nanosecond)
				event.EndTime = &t
			}
			event.Status = "resolved"
		} else {
			t := month.AddDate(0, 1, 0).Add(-time.Nanosecond)
			event.EndTime = &t
			event.Status = "ongoing"
		}

		// Calculate duration
		duration := event.EndTime.Sub(event.StartTime)
		event.Duration = formatDuration(duration)

		events = append(events, event)
	}

	return events
}

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
