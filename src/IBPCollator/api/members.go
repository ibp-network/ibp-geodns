package api

import (
	"net/http"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

type MemberInfo struct {
	Name        string   `json:"name"`
	Website     string   `json:"website,omitempty"`
	Logo        string   `json:"logo,omitempty"`
	Level       int      `json:"level"`
	JoinedDate  string   `json:"joined_date"`
	Region      string   `json:"region"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	ServiceIPv4 string   `json:"service_ipv4,omitempty"`
	ServiceIPv6 string   `json:"service_ipv6,omitempty"`
	Services    []string `json:"services"`
	Active      bool     `json:"active"`
	Override    bool     `json:"override"`
}

func handleMembers(w http.ResponseWriter, r *http.Request) {
	c := cfg.GetConfig()
	memberName := r.URL.Query().Get("name")

	if memberName != "" {
		// Get specific member
		member, exists := c.Members[memberName]
		if !exists {
			writeError(w, http.StatusNotFound, "Member not found")
			return
		}

		info := buildMemberInfo(memberName, member)
		writeJSON(w, http.StatusOK, info)
		return
	}

	// Get all members
	var members []MemberInfo
	for name, member := range c.Members {
		members = append(members, buildMemberInfo(name, member))
	}

	writeJSON(w, http.StatusOK, members)
}

func handleMemberStats(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	memberName := r.URL.Query().Get("name")
	if memberName == "" {
		writeError(w, http.StatusBadRequest, "Member name required")
		return
	}

	// Get the member's Details.Name for database lookup
	c := cfg.GetConfig()
	dbMemberName := memberName
	if member, exists := c.Members[memberName]; exists && member.Details.Name != "" {
		dbMemberName = member.Details.Name
	}

	// Get request stats
	var totalRequests int
	err = data2.DB.QueryRow(`
		SELECT COALESCE(SUM(hits), 0)
		FROM requests
		WHERE member_name = ?
		AND date >= ? AND date <= ?
	`, dbMemberName, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&totalRequests)

	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to get member requests: %v", err)
		totalRequests = 0
	}

	// Get downtime stats
	var totalDowntimeEvents int
	var totalDowntimeHours float64

	rows, err := data2.DB.Query(`
		SELECT 
			start_time,
			COALESCE(end_time, ?) as end_time
		FROM member_events
		WHERE member_name = ?
		AND status = 0
		AND start_time <= ?
		AND (end_time IS NULL OR end_time >= ?)
	`, time.Now().UTC(), dbMemberName, end, start)

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

			totalDowntimeHours += eventEnd.Sub(eventStart).Hours()
			totalDowntimeEvents++
		}
	}

	// Calculate uptime percentage
	totalHours := end.Sub(start).Hours()
	uptimePercentage := 100.0
	if totalHours > 0 {
		uptimePercentage = ((totalHours - totalDowntimeHours) / totalHours) * 100
	}

	// Get top countries
	type CountryStat struct {
		Country  string `json:"country"`
		Name     string `json:"name"`
		Requests int    `json:"requests"`
	}

	var topCountries []CountryStat
	countryRows, err := data2.DB.Query(`
		SELECT 
			country_code,
			MAX(country_name) as country_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE member_name = ?
		AND date >= ? AND date <= ?
		GROUP BY country_code
		ORDER BY total_hits DESC
		LIMIT 10
	`, dbMemberName, start.Format("2006-01-02"), end.Format("2006-01-02"))

	if err == nil {
		defer countryRows.Close()
		for countryRows.Next() {
			var stat CountryStat
			countryRows.Scan(&stat.Country, &stat.Name, &stat.Requests)
			topCountries = append(topCountries, stat)
		}
	}

	// Get service breakdown
	type ServiceStat struct {
		Service  string `json:"service"`
		Domain   string `json:"domain"`
		Requests int    `json:"requests"`
	}

	var serviceStats []ServiceStat
	serviceRows, err := data2.DB.Query(`
		SELECT 
			domain_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE member_name = ?
		AND date >= ? AND date <= ?
		AND domain_name != ''
		GROUP BY domain_name
		ORDER BY total_hits DESC
	`, dbMemberName, start.Format("2006-01-02"), end.Format("2006-01-02"))

	if err == nil {
		defer serviceRows.Close()
		for serviceRows.Next() {
			var stat ServiceStat
			serviceRows.Scan(&stat.Domain, &stat.Requests)
			stat.Service = domainToServiceName(stat.Domain)
			serviceStats = append(serviceStats, stat)
		}
	}

	stats := map[string]interface{}{
		"member_name":           memberName,
		"start_date":            start.Format("2006-01-02"),
		"end_date":              end.Format("2006-01-02"),
		"total_requests":        totalRequests,
		"total_downtime_events": totalDowntimeEvents,
		"total_downtime_hours":  totalDowntimeHours,
		"uptime_percentage":     uptimePercentage,
		"top_countries":         topCountries,
		"service_breakdown":     serviceStats,
	}

	writeJSON(w, http.StatusOK, stats)
}

func buildMemberInfo(name string, member cfg.Member) MemberInfo {
	info := MemberInfo{
		Name:        name,
		Website:     member.Details.Website,
		Logo:        member.Details.Logo,
		Level:       member.Membership.Level,
		JoinedDate:  time.Unix(int64(member.Membership.Joined), 0).Format("2006-01-02"),
		Region:      member.Location.Region,
		Latitude:    member.Location.Latitude,
		Longitude:   member.Location.Longitude,
		ServiceIPv4: member.Service.ServiceIPv4,
		ServiceIPv6: member.Service.ServiceIPv6,
		Active:      member.Service.Active == 1,
		Override:    member.Override,
		Services:    []string{},
	}

	// Collect all services
	for _, svcList := range member.ServiceAssignments {
		info.Services = append(info.Services, svcList...)
	}

	return info
}
