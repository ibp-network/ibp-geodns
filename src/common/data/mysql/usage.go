package mysql

import (
	"fmt"
	"time"
)

// UsageAggregate represents aggregated usage count grouped by a key
// such as domain, member or country.
type UsageAggregate struct {
	Key      string
	Requests int
}

// GetUsageByDomain returns total request counts grouped by domain
// between the provided start and end times.
func GetUsageByDomain(start, end time.Time) ([]UsageAggregate, error) {
	query := `
        SELECT domain_name AS k, COUNT(*) AS requests
        FROM usage_records
        WHERE request_time BETWEEN ? AND ?
        GROUP BY domain_name
        ORDER BY requests DESC
    `
	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query usage by domain: %w", err)
	}
	defer rows.Close()

	var results []UsageAggregate
	for rows.Next() {
		var r UsageAggregate
		if err := rows.Scan(&r.Key, &r.Requests); err != nil {
			return nil, fmt.Errorf("scan usage by domain: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByMember returns total request counts grouped by member name
// between the provided start and end times.
func GetUsageByMember(start, end time.Time) ([]UsageAggregate, error) {
	query := `
        SELECT member_name AS k, COUNT(*) AS requests
        FROM usage_records
        WHERE request_time BETWEEN ? AND ?
        GROUP BY member_name
        ORDER BY requests DESC
    `
	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query usage by member: %w", err)
	}
	defer rows.Close()

	var results []UsageAggregate
	for rows.Next() {
		var r UsageAggregate
		if err := rows.Scan(&r.Key, &r.Requests); err != nil {
			return nil, fmt.Errorf("scan usage by member: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByCountry returns request counts grouped by country code
// between the provided start and end times.
func GetUsageByCountry(start, end time.Time) ([]UsageAggregate, error) {
	query := `
        SELECT country_code AS k, COUNT(*) AS requests
        FROM usage_records
        WHERE request_time BETWEEN ? AND ?
        GROUP BY country_code
        ORDER BY requests DESC
    `
	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query usage by country: %w", err)
	}
	defer rows.Close()

	var results []UsageAggregate
	for rows.Next() {
		var r UsageAggregate
		if err := rows.Scan(&r.Key, &r.Requests); err != nil {
			return nil, fmt.Errorf("scan usage by country: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByClassC returns request counts grouped by Class C network
// between the provided start and end times.
func GetUsageByClassC(start, end time.Time) ([]UsageAggregate, error) {
	query := `
        SELECT class_c AS k, COUNT(*) AS requests
        FROM usage_records
        WHERE request_time BETWEEN ? AND ?
        GROUP BY class_c
        ORDER BY requests DESC
    `
	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query usage by class c: %w", err)
	}
	defer rows.Close()

	var results []UsageAggregate
	for rows.Next() {
		var r UsageAggregate
		if err := rows.Scan(&r.Key, &r.Requests); err != nil {
			return nil, fmt.Errorf("scan usage by class c: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}
