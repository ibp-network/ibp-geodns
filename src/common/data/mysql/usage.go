package mysql

import (
	"fmt"
	"time"
)

// UsageRecord represents a row in the daily_usage table.
type UsageRecord struct {
	Date    time.Time
	Domain  string
	ASN     string
	Subnet  string
	Country string
	Hits    int
}

// InsertOrUpdateUsage inserts a new usage record or increments hits if it already exists.
func InsertOrUpdateUsage(record UsageRecord) error {
	query := `
                INSERT INTO daily_usage (date, domain, asn, subnet, country, hits)
                VALUES (?, ?, ?, ?, ?, ?)
                ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
        `

	_, err := DB.Exec(query, record.Date, record.Domain, record.ASN, record.Subnet, record.Country, record.Hits)
	if err != nil {
		return fmt.Errorf("failed to insert/update usage: %w", err)
	}

	return nil
}

// QueryUsageByDomain returns aggregated hits grouped by domain for the range.
func QueryUsageByDomain(start, end time.Time) (map[string]int, error) {
	query := `
                SELECT domain, SUM(hits) as total
                FROM daily_usage
                WHERE date >= ? AND date <= ?
                GROUP BY domain
        `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query by domain failed: %w", err)
	}
	defer rows.Close()

	results := make(map[string]int)
	for rows.Next() {
		var domain string
		var total int
		if err := rows.Scan(&domain, &total); err != nil {
			return nil, fmt.Errorf("scan domain usage: %w", err)
		}
		results[domain] = total
	}

	return results, nil
}

// QueryUsageByMember aggregates hits by member for the range. The member is
// looked up based on the domain column using a configured mapping in the
// database.
func QueryUsageByMember(start, end time.Time) (map[string]int, error) {
	query := `
                SELECT m.member_name, SUM(u.hits) as total
                FROM daily_usage u
                JOIN domains m ON u.domain = m.domain
                WHERE u.date >= ? AND u.date <= ?
                GROUP BY m.member_name
        `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query by member failed: %w", err)
	}
	defer rows.Close()

	results := make(map[string]int)
	for rows.Next() {
		var member string
		var total int
		if err := rows.Scan(&member, &total); err != nil {
			return nil, fmt.Errorf("scan member usage: %w", err)
		}
		results[member] = total
	}

	return results, nil
}

// QueryUsageByCountry aggregates hits by country for the range.
func QueryUsageByCountry(start, end time.Time) (map[string]int, error) {
	query := `
                SELECT country, SUM(hits) as total
                FROM daily_usage
                WHERE date >= ? AND date <= ?
                GROUP BY country
        `

	rows, err := DB.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query by country failed: %w", err)
	}
	defer rows.Close()

	results := make(map[string]int)
	for rows.Next() {
		var country string
		var total int
		if err := rows.Scan(&country, &total); err != nil {
			return nil, fmt.Errorf("scan country usage: %w", err)
		}
		results[country] = total
	}

	return results, nil
}
