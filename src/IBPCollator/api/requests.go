package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
)

type RequestFilter struct {
	Country string
	ASN     string
	Network string
	Service string
	Member  string
	Domain  string
}

type RequestStats struct {
	Date        string `json:"date"`
	Country     string `json:"country,omitempty"`
	CountryName string `json:"country_name,omitempty"`
	ASN         string `json:"asn,omitempty"`
	Network     string `json:"network,omitempty"`
	Service     string `json:"service,omitempty"`
	Member      string `json:"member,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Requests    int    `json:"requests"`
}

func parseRequestFilters(r *http.Request) (RequestFilter, error) {
	filter := RequestFilter{
		Country: r.URL.Query().Get("country"),
		ASN:     r.URL.Query().Get("asn"),
		Network: r.URL.Query().Get("network"),
		Service: r.URL.Query().Get("service"),
		Member:  r.URL.Query().Get("member"),
		Domain:  r.URL.Query().Get("domain"),
	}

	// Validate and sanitize the filter
	if err := sanitizeRequestFilter(&filter); err != nil {
		return filter, err
	}

	return filter, nil
}

func handleRequestsByCountry(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	filters, err := parseRequestFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid filter: %v", err))
		return
	}

	query := `
		SELECT 
			date,
			country_code,
			MAX(country_name) as country_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	args := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Apply filters with parameterized queries
	if filters.Member != "" {
		query += " AND member_name = ?"
		args = append(args, filters.Member)
	}
	if filters.Domain != "" {
		query += " AND domain_name = ?"
		args = append(args, filters.Domain)
	}
	if filters.ASN != "" {
		query += " AND network_asn = ?"
		args = append(args, filters.ASN)
	}
	if filters.Network != "" {
		query += " AND network_name LIKE ?"
		args = append(args, "%"+filters.Network+"%")
	}

	query += " GROUP BY date, country_code ORDER BY date, total_hits DESC"

	rows, err := data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query requests by country: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var results []RequestStats
	for rows.Next() {
		var stat RequestStats
		var countryName sql.NullString

		err := rows.Scan(&stat.Date, &stat.Country, &countryName, &stat.Requests)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan row: %v", err)
			continue
		}

		if countryName.Valid {
			stat.CountryName = countryName.String
		}

		results = append(results, stat)
	}

	writeJSON(w, http.StatusOK, results)
}

func handleRequestsByASN(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	filters, err := parseRequestFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid filter: %v", err))
		return
	}

	query := `
		SELECT 
			date,
			COALESCE(network_asn, 'Unknown') as asn,
			COALESCE(network_name, 'Unknown') as network,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	args := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Apply filters
	if filters.Country != "" {
		query += " AND country_code = ?"
		args = append(args, filters.Country)
	}
	if filters.Member != "" {
		query += " AND member_name = ?"
		args = append(args, filters.Member)
	}
	if filters.Domain != "" {
		query += " AND domain_name = ?"
		args = append(args, filters.Domain)
	}

	query += " GROUP BY date, network_asn, network_name ORDER BY date, total_hits DESC"

	rows, err := data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query requests by ASN: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var results []RequestStats
	for rows.Next() {
		var stat RequestStats
		err := rows.Scan(&stat.Date, &stat.ASN, &stat.Network, &stat.Requests)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan row: %v", err)
			continue
		}
		results = append(results, stat)
	}

	writeJSON(w, http.StatusOK, results)
}

func handleRequestsByService(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	filters, err := parseRequestFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid filter: %v", err))
		return
	}

	query := `
		SELECT 
			date,
			domain_name as domain,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
		AND domain_name != ''
	`

	args := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Apply filters
	if filters.Country != "" {
		query += " AND country_code = ?"
		args = append(args, filters.Country)
	}
	if filters.Member != "" {
		query += " AND member_name = ?"
		args = append(args, filters.Member)
	}
	if filters.ASN != "" {
		query += " AND network_asn = ?"
		args = append(args, filters.ASN)
	}

	query += " GROUP BY date, domain_name ORDER BY date, total_hits DESC"

	rows, err := data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query requests by service: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var results []RequestStats
	for rows.Next() {
		var stat RequestStats
		err := rows.Scan(&stat.Date, &stat.Domain, &stat.Requests)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan row: %v", err)
			continue
		}

		// Map domain to service name
		stat.Service = domainToServiceName(stat.Domain)

		results = append(results, stat)
	}

	writeJSON(w, http.StatusOK, results)
}

func handleRequestsByMember(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	filters, err := parseRequestFilters(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid filter: %v", err))
		return
	}

	query := `
		SELECT 
			date,
			COALESCE(member_name, '(none)') as member,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	args := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Apply filters
	if filters.Country != "" {
		query += " AND country_code = ?"
		args = append(args, filters.Country)
	}
	if filters.Domain != "" {
		query += " AND domain_name = ?"
		args = append(args, filters.Domain)
	}
	if filters.ASN != "" {
		query += " AND network_asn = ?"
		args = append(args, filters.ASN)
	}

	query += " GROUP BY date, member_name ORDER BY date, total_hits DESC"

	rows, err := data2.DB.Query(query, args...)
	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to query requests by member: %v", err)
		writeError(w, http.StatusInternalServerError, "Database error")
		return
	}
	defer rows.Close()

	var results []RequestStats
	for rows.Next() {
		var stat RequestStats
		err := rows.Scan(&stat.Date, &stat.Member, &stat.Requests)
		if err != nil {
			log.Log(log.Error, "[CollatorAPI] Failed to scan row: %v", err)
			continue
		}
		results = append(results, stat)
	}

	writeJSON(w, http.StatusOK, results)
}

func handleRequestsSummary(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseTimeParams(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Validate dates
	if !validateDate(start.Format("2006-01-02")) || !validateDate(end.Format("2006-01-02")) {
		writeError(w, http.StatusBadRequest, "Invalid date format")
		return
	}

	// Get total requests
	var totalRequests int
	err = data2.DB.QueryRow(`
		SELECT COALESCE(SUM(hits), 0) 
		FROM requests 
		WHERE date >= ? AND date <= ?
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&totalRequests)

	if err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to get total requests: %v", err)
		totalRequests = 0
	}

	// Get unique counts
	var uniqueCountries, uniqueASNs, uniqueMembers, uniqueDomains int

	data2.DB.QueryRow(`
		SELECT COUNT(DISTINCT country_code) 
		FROM requests 
		WHERE date >= ? AND date <= ?
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&uniqueCountries)

	data2.DB.QueryRow(`
		SELECT COUNT(DISTINCT network_asn) 
		FROM requests 
		WHERE date >= ? AND date <= ? AND network_asn IS NOT NULL
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&uniqueASNs)

	data2.DB.QueryRow(`
		SELECT COUNT(DISTINCT member_name) 
		FROM requests 
		WHERE date >= ? AND date <= ? AND member_name IS NOT NULL
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&uniqueMembers)

	data2.DB.QueryRow(`
		SELECT COUNT(DISTINCT domain_name) 
		FROM requests 
		WHERE date >= ? AND date <= ? AND domain_name != ''
	`, start.Format("2006-01-02"), end.Format("2006-01-02")).Scan(&uniqueDomains)

	summary := map[string]interface{}{
		"start_date":       start.Format("2006-01-02"),
		"end_date":         end.Format("2006-01-02"),
		"total_requests":   totalRequests,
		"unique_countries": uniqueCountries,
		"unique_asns":      uniqueASNs,
		"unique_members":   uniqueMembers,
		"unique_domains":   uniqueDomains,
	}

	writeJSON(w, http.StatusOK, summary)
}

// Helper function to convert domain to service name
func domainToServiceName(domain string) string {
	// First try to find exact match in config
	c := cfg.GetConfig()
	for serviceName, service := range c.Services {
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if strings.Contains(strings.ToLower(rpcUrl), strings.ToLower(domain)) {
					return serviceName
				}
			}
		}
	}

	// Fallback: clean up the domain name
	name := strings.TrimSuffix(domain, ".dotters.network")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, ".", " ")

	// Title case each word
	parts := strings.Fields(name)
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(parts, " ")
}
