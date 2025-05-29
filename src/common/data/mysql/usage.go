package mysql

import (
	"database/sql"
	"fmt"
)

// UsageRecord remains the same. We note that MemberName, Asn, NetworkName, CountryName
// can hold the expanded columns from the DB.
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
	_, err := DB.Exec(query,
		rec.Date,
		rec.Domain,
		rec.MemberName,
		rec.CountryCode,
		rec.Hits,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert usage record: %w", err)
	}
	return nil
}

// GetUsageByDomain returns aggregated usage grouped by country, member_name, asn, network_name, country_name, etc.
func GetUsageByDomain(domain, startDate, endDate string) ([]UsageRecord, error) {
	// Expanded query to fetch member_name, asn, network_name, country_name if they exist in usage_daily.
	// If your table has those columns, incorporate them. If not, adapt accordingly.
	query := `
        SELECT
            usage_date,
            domain,
            IFNULL(member_name, '') AS member_name,
            country_code,
            SUM(hits) as hits
        FROM usage_daily
        WHERE domain = ?
          AND usage_date BETWEEN ? AND ?
        GROUP BY usage_date, domain, member_name, country_code
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
		// We'll scan into the usage_date, domain, member_name, country_code, hits
		// The query uses IFNULL(member_name, '') to ensure no NULL returned, but we'll keep it in NullString
		err := rows.Scan(&r.Date, &r.Domain, &r.MemberName, &r.CountryCode, &r.Hits)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}

// GetUsageByMember returns usage grouped by country, as well as extended columns.
func GetUsageByMember(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	query := `
        SELECT
            usage_date,
            domain,
            IFNULL(member_name, '') AS member_name,
            country_code,
            SUM(hits) as hits
        FROM usage_daily
        WHERE domain = ?
          AND member_name = ?
          AND usage_date BETWEEN ? AND ?
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
		err := rows.Scan(&r.Date, &r.Domain, &r.MemberName, &r.CountryCode, &r.Hits)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}

// GetUsageByCountry returns overall usage grouped by country. We could also expand to asn, network_name, etc.
func GetUsageByCountry(startDate, endDate string) ([]UsageRecord, error) {
	query := `
        SELECT
            usage_date,
            domain,
            IFNULL(member_name, '') AS member_name,
            country_code,
            SUM(hits) as hits
        FROM usage_daily
        WHERE usage_date BETWEEN ? AND ?
        GROUP BY usage_date, domain, member_name, country_code
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
		err := rows.Scan(&r.Date, &r.Domain, &r.MemberName, &r.CountryCode, &r.Hits)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}
