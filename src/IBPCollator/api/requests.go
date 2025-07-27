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
	Countries []string
	ASNs      []string
	Networks  []string
	Services  []string
	Members   []string
	Domains   []string
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
		Countries: parseMultiValue(r.URL.Query().Get("country")),
		ASNs:      parseMultiValue(r.URL.Query().Get("asn")),
		Networks:  parseMultiValue(r.URL.Query().Get("network")),
		Services:  parseMultiValue(r.URL.Query().Get("service")),
		Members:   parseMultiValue(r.URL.Query().Get("member")),
		Domains:   parseMultiValue(r.URL.Query().Get("domain")),
	}

	// Validate and sanitize the filter
	if err := sanitizeRequestFilter(&filter); err != nil {
		return filter, err
	}

	return filter, nil
}

// parseMultiValue splits comma-separated values and trims whitespace
func parseMultiValue(value string) []string {
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// buildFilterConditions builds SQL WHERE conditions and args for filters
func buildFilterConditions(filter RequestFilter, baseArgs []interface{}) (string, []interface{}) {
	conditions := []string{}
	args := append([]interface{}{}, baseArgs...)

	if len(filter.Countries) > 0 {
		placeholders := make([]string, len(filter.Countries))
		for i, country := range filter.Countries {
			placeholders[i] = "?"
			args = append(args, country)
		}
		conditions = append(conditions, fmt.Sprintf("country_code IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.ASNs) > 0 {
		placeholders := make([]string, len(filter.ASNs))
		for i, asn := range filter.ASNs {
			placeholders[i] = "?"
			args = append(args, asn)
		}
		conditions = append(conditions, fmt.Sprintf("network_asn IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.Networks) > 0 {
		networkConditions := make([]string, len(filter.Networks))
		for i, network := range filter.Networks {
			networkConditions[i] = "network_name LIKE ?"
			args = append(args, "%"+network+"%")
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(networkConditions, " OR ")))
	}

	if len(filter.Members) > 0 {
		placeholders := make([]string, len(filter.Members))
		for i, member := range filter.Members {
			placeholders[i] = "?"
			args = append(args, member)
		}
		conditions = append(conditions, fmt.Sprintf("member_name IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filter.Domains) > 0 {
		placeholders := make([]string, len(filter.Domains))
		for i, domain := range filter.Domains {
			placeholders[i] = "?"
			args = append(args, domain)
		}
		conditions = append(conditions, fmt.Sprintf("domain_name IN (%s)", strings.Join(placeholders, ",")))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
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

	baseQuery := `
		SELECT 
			date,
			country_code,
			MAX(country_name) as country_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	baseArgs := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Build filter conditions
	whereClause, args := buildFilterConditions(filters, baseArgs)
	query := baseQuery + whereClause + " GROUP BY date, country_code ORDER BY date, total_hits DESC"

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

	baseQuery := `
		SELECT 
			date,
			COALESCE(network_asn, 'Unknown') as asn,
			COALESCE(network_name, 'Unknown') as network,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	baseArgs := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Build filter conditions
	whereClause, args := buildFilterConditions(filters, baseArgs)
	query := baseQuery + whereClause + " GROUP BY date, network_asn, network_name ORDER BY date, total_hits DESC"

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

	baseQuery := `
		SELECT 
			date,
			domain_name as domain,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
		AND domain_name != ''
	`

	baseArgs := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Build filter conditions but don't use service filter directly
	// Instead, convert services to domain patterns
	whereClause := ""
	args := baseArgs

	// Handle service filtering specially
	if len(filters.Services) > 0 {
		// Convert service names to domain patterns
		domainConditions := []string{}
		for _, service := range filters.Services {
			// Create pattern for domain matching (case-insensitive)
			domainConditions = append(domainConditions, "LOWER(domain_name) LIKE LOWER(?)")
			args = append(args, "%"+strings.ReplaceAll(strings.ToLower(service), " ", "-")+"%")
		}
		whereClause += " AND (" + strings.Join(domainConditions, " OR ") + ")"
	}

	// Add other filters
	otherFilters := RequestFilter{
		Countries: filters.Countries,
		ASNs:      filters.ASNs,
		Networks:  filters.Networks,
		Members:   filters.Members,
		Domains:   filters.Domains,
	}
	otherWhere, otherArgs := buildFilterConditions(otherFilters, []interface{}{})
	if otherWhere != "" {
		whereClause += otherWhere
		args = append(args, otherArgs...)
	}

	query := baseQuery + whereClause + " GROUP BY date, domain_name ORDER BY date, total_hits DESC"

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

	baseQuery := `
		SELECT 
			date,
			COALESCE(member_name, '(none)') as member,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
	`

	baseArgs := []interface{}{start.Format("2006-01-02"), end.Format("2006-01-02")}

	// Build filter conditions
	whereClause, args := buildFilterConditions(filters, baseArgs)
	query := baseQuery + whereClause + " GROUP BY date, member_name ORDER BY date, total_hits DESC"

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

// Helper function to convert service names to domains
func convertServicesToDomains(services []string) []string {
	c := cfg.GetConfig()
	domains := []string{}

	for _, serviceName := range services {
		// Try to find matching domains for this service
		for svcName, svc := range c.Services {
			if strings.EqualFold(svcName, serviceName) {
				for _, provider := range svc.Providers {
					for _, rpcUrl := range provider.RpcUrls {
						if domain := extractDomainFromURL(rpcUrl); domain != "" {
							domains = append(domains, domain)
						}
					}
				}
			}
		}
	}

	return domains
}

// Helper function to convert domain to service name with improved matching
func domainToServiceName(domain string) string {
	// First try to find exact match in config
	c := cfg.GetConfig()
	domainLower := strings.ToLower(domain)

	// Clean up domain for comparison
	cleanDomain := strings.TrimSuffix(domainLower, ".dotters.network")
	cleanDomain = strings.TrimSuffix(cleanDomain, ".ibp.network")

	for serviceName, service := range c.Services {
		serviceNameLower := strings.ToLower(serviceName)

		// Check for exact match after cleaning
		if cleanDomain == serviceNameLower {
			return serviceName
		}

		// Check if any RPC URL contains this exact domain
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				rpcUrlLower := strings.ToLower(rpcUrl)
				if strings.Contains(rpcUrlLower, domainLower) {
					return serviceName
				}
			}
		}
	}

	// Fallback: clean up the domain name for display
	name := strings.TrimSuffix(domain, ".dotters.network")
	name = strings.TrimSuffix(name, ".ibp.network")

	// Don't replace hyphens in the middle of service names
	// This prevents "eth-asset-hub-paseo" from becoming "Eth Asset Hub Paseo"
	// Only capitalize first letter of each hyphenated part
	parts := strings.Split(name, "-")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + strings.ToLower(part[1:])
		}
	}

	return strings.Join(parts, "-")
}
