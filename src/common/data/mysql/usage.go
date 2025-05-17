package mysql

import (
	"database/sql"
	"fmt"
)

type UsageRecord struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Hits        int
}

// UpsertUsageRecord inserts or updates a usage record.
func UpsertUsageRecord(rec UsageRecord) error {
	query := `
        INSERT INTO usage_daily
            (usage_date, domain, member_name, country_code, hits)
        VALUES (?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
    `
	_, err := DB.Exec(query, rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
	if err != nil {
		return fmt.Errorf("failed to upsert usage record: %w", err)
	}
	return nil
}

// GetUsageByDomain returns aggregated usage grouped by country for a domain.
func GetUsageByDomain(domain, startDate, endDate string) ([]UsageRecord, error) {
	query := `
        SELECT usage_date, domain, country_code, SUM(hits) as hits
        FROM usage_daily
        WHERE domain = ? AND usage_date BETWEEN ? AND ?
        GROUP BY usage_date, domain, country_code
        ORDER BY usage_date
    `
	rows, err := DB.Query(query, domain, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var res []UsageRecord
	for rows.Next() {
		var r UsageRecord
		if err := rows.Scan(&r.Date, &r.Domain, &r.CountryCode, &r.Hits); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}

// GetUsageByMember returns usage grouped by country for a domain/member.
func GetUsageByMember(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	query := `
        SELECT usage_date, domain, member_name, country_code, SUM(hits) as hits
        FROM usage_daily
        WHERE domain = ? AND member_name = ? AND usage_date BETWEEN ? AND ?
        GROUP BY usage_date, domain, member_name, country_code
        ORDER BY usage_date
    `
	rows, err := DB.Query(query, domain, member, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var res []UsageRecord
	for rows.Next() {
		var r UsageRecord
		if err := rows.Scan(&r.Date, &r.Domain, &r.MemberName, &r.CountryCode, &r.Hits); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}

// GetUsageByCountry returns overall usage grouped by country.
func GetUsageByCountry(startDate, endDate string) ([]UsageRecord, error) {
	query := `
        SELECT usage_date, country_code, SUM(hits) as hits
        FROM usage_daily
        WHERE usage_date BETWEEN ? AND ?
        GROUP BY usage_date, country_code
        ORDER BY usage_date
    `
	rows, err := DB.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var res []UsageRecord
	for rows.Next() {
		var r UsageRecord
		if err := rows.Scan(&r.Date, &r.CountryCode, &r.Hits); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}
