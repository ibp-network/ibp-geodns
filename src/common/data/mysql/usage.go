package mysql

import (
	"database/sql"
	"fmt"
)

// UsageRecord now includes all fields that we store in usage_daily / usage_daily_v6.
type UsageRecord struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Asn         sql.NullString
	NetworkName sql.NullString
	CountryName sql.NullString
	Hits        int
}

// ------------------------------------------------------------------------
// IPv4 - usage_daily
// ------------------------------------------------------------------------

// UpsertUsageRecord inserts/updates a record in usage_daily.
func UpsertUsageRecord(rec UsageRecord) error {
	q := `
INSERT INTO usage_daily
  (usage_date, domain, member_name, country_code, asn, network_name, country_name, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  hits = hits + VALUES(hits)
`
	_, err := DB.Exec(
		q,
		rec.Date,
		rec.Domain,
		safeNullStr(rec.MemberName),
		rec.CountryCode,
		safeNullStr(rec.Asn),
		safeNullStr(rec.NetworkName),
		safeNullStr(rec.CountryName),
		rec.Hits,
	)
	if err != nil {
		return fmt.Errorf("failed UpsertUsageRecord(v4): %w", err)
	}
	return nil
}

// GetUsageByDomain returns IPv4 usage records for a domain in [startDate, endDate].
func GetUsageByDomain(domain, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily
WHERE domain = ?
  AND usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, domain, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByDomain(v4) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByDomain(v4) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByMember returns IPv4 usage for a given domain+member in [startDate, endDate].
func GetUsageByMember(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily
WHERE domain = ?
  AND member_name = ?
  AND usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, domain, member, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByMember(v4) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByMember(v4) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByCountry returns IPv4 usage in [startDate, endDate], grouped by date/domain/country/etc.
func GetUsageByCountry(startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily
WHERE usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByCountry(v4) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByCountry(v4) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// ------------------------------------------------------------------------
// IPv6 - usage_daily_v6
// ------------------------------------------------------------------------

// UpsertUsageRecordV6 inserts/updates a record in usage_daily_v6.
func UpsertUsageRecordV6(rec UsageRecord) error {
	q := `
INSERT INTO usage_daily_v6
  (usage_date, domain, member_name, country_code, asn, network_name, country_name, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  hits = hits + VALUES(hits)
`
	_, err := DB.Exec(
		q,
		rec.Date,
		rec.Domain,
		safeNullStr(rec.MemberName),
		rec.CountryCode,
		safeNullStr(rec.Asn),
		safeNullStr(rec.NetworkName),
		safeNullStr(rec.CountryName),
		rec.Hits,
	)
	if err != nil {
		return fmt.Errorf("failed UpsertUsageRecord(v6): %w", err)
	}
	return nil
}

// GetUsageByDomainV6 returns IPv6 usage records for a domain in [startDate, endDate].
func GetUsageByDomainV6(domain, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily_v6
WHERE domain = ?
  AND usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, domain, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByDomain(v6) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByDomain(v6) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByMemberV6 returns IPv6 usage for a given domain+member in [startDate, endDate].
func GetUsageByMemberV6(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily_v6
WHERE domain = ?
  AND member_name = ?
  AND usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, domain, member, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByMember(v6) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByMember(v6) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// GetUsageByCountryV6 returns IPv6 usage in [startDate, endDate], grouped by date/domain/etc.
func GetUsageByCountryV6(startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  usage_date,
  domain,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(asn,'') AS asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM usage_daily_v6
WHERE usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, member_name, country_code, asn, network_name, country_name
ORDER BY usage_date
`
	rows, err := DB.Query(q, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByCountry(v6) query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		err = rows.Scan(
			&r.Date,
			&r.Domain,
			&r.MemberName,
			&r.CountryCode,
			&r.Asn,
			&r.NetworkName,
			&r.CountryName,
			&r.Hits,
		)
		if err != nil {
			return nil, fmt.Errorf("GetUsageByCountry(v6) scan error: %w", err)
		}
		results = append(results, r)
	}
	return results, nil
}

// ------------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------------

func safeNullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
