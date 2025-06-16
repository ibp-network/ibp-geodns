package data

import (
	"database/sql"
	"fmt"
	"time"

	mysql "ibp-geodns/src/common/data/mysql"
)

type UsageRecord struct {
	Date        string
	Domain      string
	MemberName  string
	CountryCode string
	Asn         string
	NetworkName string
	CountryName string
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
	_, err := mysql.DB.Exec(
		q,
		rec.Date,
		rec.Domain,
		nullOrString(rec.MemberName),
		nullOrString(rec.CountryCode),
		nullOrString(rec.Asn),
		nullOrString(rec.NetworkName),
		nullOrString(rec.CountryName),
		rec.Hits,
	)
	if err != nil {
		return fmt.Errorf("failed UpsertUsageRecord: %w", err)
	}
	return nil
}

func GetUsageByDomain(domain string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")

	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  IFNULL(country_code,'') AS country_code,
  IFNULL(network_asn,'') as network_asn,
  IFNULL(network_name,'') as network_name,
  IFNULL(country_name,'') as country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
`
	rows, err := mysql.DB.Query(q, domain, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByDomain query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		var mName, cCode, a, netName, cName sql.NullString

		var dateStr, dom string
		var hits int

		if err := rows.Scan(&dateStr, &dom, &mName, &cCode, &a, &netName, &cName, &hits); err != nil {
			return nil, fmt.Errorf("GetUsageByDomain scan error: %w", err)
		}
		r.Date = dateStr
		r.Domain = dom
		r.MemberName = mName.String
		r.CountryCode = cCode.String
		r.Asn = a.String
		r.NetworkName = netName.String
		r.CountryName = cName.String
		r.Hits = hits

		results = append(results, r)
	}
	return results, nil
}

func GetUsageByMember(domain, member string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")

	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  IFNULL(country_code,'') as country_code,
  IFNULL(network_asn,'') as network_asn,
  IFNULL(network_name,'') as network_name,
  IFNULL(country_name,'') as country_name,
  SUM(hits) AS hits
FROM requests
WHERE domain_name = ?
  AND member_name = ?
  AND date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
`
	rows, err := mysql.DB.Query(q, domain, member, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByMember query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		var mName, cCode, a, netName, cName sql.NullString

		var dateStr, dom string
		var hits int

		if err := rows.Scan(&dateStr, &dom, &mName, &cCode, &a, &netName, &cName, &hits); err != nil {
			return nil, fmt.Errorf("GetUsageByMember scan error: %w", err)
		}
		r.Date = dateStr
		r.Domain = dom
		r.MemberName = mName.String
		r.CountryCode = cCode.String
		r.Asn = a.String
		r.NetworkName = netName.String
		r.CountryName = cName.String
		r.Hits = hits

		results = append(results, r)
	}
	return results, nil
}

func GetUsageByCountry(start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")

	q := `
SELECT
  date,
  domain_name,
  IFNULL(member_name,'') AS member_name,
  IFNULL(country_code,'') as country_code,
  IFNULL(network_asn,'') as network_asn,
  IFNULL(network_name,'') as network_name,
  IFNULL(country_name,'') as country_name,
  SUM(hits) AS hits
FROM requests
WHERE date BETWEEN ? AND ?
GROUP BY date, domain_name, member_name, country_code, network_asn, network_name, country_name
ORDER BY date
`
	rows, err := mysql.DB.Query(q, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("GetUsageByCountry query error: %w", err)
	}
	defer rows.Close()

	var results []UsageRecord
	for rows.Next() {
		var r UsageRecord
		var mName, cCode, a, netName, cName sql.NullString

		var dateStr, dom string
		var hits int

		if err := rows.Scan(&dateStr, &dom, &mName, &cCode, &a, &netName, &cName, &hits); err != nil {
			return nil, fmt.Errorf("GetUsageByCountry scan error: %w", err)
		}
		r.Date = dateStr
		r.Domain = dom
		r.MemberName = mName.String
		r.CountryCode = cCode.String
		r.Asn = a.String
		r.NetworkName = netName.String
		r.CountryName = cName.String
		r.Hits = hits

		results = append(results, r)
	}
	return results, nil
}

func nullOrString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
