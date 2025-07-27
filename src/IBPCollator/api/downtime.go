package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

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
