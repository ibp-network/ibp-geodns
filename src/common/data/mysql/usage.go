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
	Asn         sql.NullString
	NetworkName sql.NullString
	CountryName sql.NullString
	Hits        int
}

func UpsertUsageRecord(rec UsageRecord) error {
	q := `
INSERT INTO requests
  (date, domain_name, member_name, country_code, network_asn, network_name, country_name, is_ipv6, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, '0', ?)
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

func GetUsageByDomain(domain, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func GetUsageByMember(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND member_name = ?
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func GetUsageByCountry(startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func UpsertUsageRecordV6(rec UsageRecord) error {
	q := `
INSERT INTO requests
  (date, domain_name, member_name, country_code, network_asn, network_name, country_name, is_ipv6, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, '1', ?)
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

func GetUsageByDomainV6(domain, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND is_ipv6 = '1' 
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func GetUsageByMemberV6(domain, member, startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND member_name = ?
  AND is_ipv6 = '1'
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func GetUsageByCountryV6(startDate, endDate string) ([]UsageRecord, error) {
	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  country_code,
  IFNULL(network_asn,'') AS network_asn,
  IFNULL(network_name,'') AS network_name,
  IFNULL(country_name,'') AS country_name,
  SUM(hits) AS hits
FROM requests
WHERE is_ipv6 = '1' 
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
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

func safeNullStr(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}
