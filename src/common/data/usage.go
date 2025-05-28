package data

import (
	"database/sql"
	"fmt"

	mysql "ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
	"time"
)

// usage.go for IPv4 usage
// The below functions remain for usage_daily table

type UsageRecord struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Asn         sql.NullString // new field for ASN
	NetworkName sql.NullString // new field for network_name
	CountryName sql.NullString // new field for country_name
	Hits        int
}

// UpsertUsageRecord inserts or updates a usage record for IPv4
func UpsertUsageRecord(rec UsageRecord) error {
	// If any of these are NULL, store empty string to avoid MySQL unique-index duplication with NULL
	dt := rec.Date
	dm := rec.Domain
	mn := safeStringOrEmpty(rec.MemberName)
	cc := rec.CountryCode
	a := safeStringOrEmpty(rec.Asn)
	nw := safeStringOrEmpty(rec.NetworkName)
	cn := safeStringOrEmpty(rec.CountryName)
	h := rec.Hits

	query := `
INSERT INTO usage_daily
(usage_date, domain, member_name, country_code, asn, network_name, country_name, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
`
	_, err := mysql.DB.Exec(query, dt, dm, mn, cc, a, nw, cn, h)
	if err != nil {
		return fmt.Errorf("failed to upsert usage record: %w", err)
	}
	return nil
}

// GetUsageByDomain returns aggregated usage grouped by country for a domain.
func GetUsageByDomain(domain string, start, end time.Time) ([]mysql.UsageRecord, error) {
	return mysql.GetUsageByDomain(domain, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func GetUsageByMember(domain, member string, start, end time.Time) ([]mysql.UsageRecord, error) {
	return mysql.GetUsageByMember(domain, member, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func GetUsageByCountry(start, end time.Time) ([]mysql.UsageRecord, error) {
	return mysql.GetUsageByCountry(start.Format("2006-01-02"), end.Format("2006-01-02"))
}

// ----------------------------------------------------------------
// IPv6 usage for usage_daily_v6
// ----------------------------------------------------------------

type UsageRecordV6 struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Asn         sql.NullString // new field for ASN
	NetworkName sql.NullString // new field for network_name
	CountryName sql.NullString // new field for country_name
	Hits        int
}

func UpsertUsageRecordV6(rec UsageRecordV6) error {
	// Same approach: ensure no NULL columns in the unique index
	dt := rec.Date
	dm := rec.Domain
	mn := safeStringOrEmpty(rec.MemberName)
	cc := rec.CountryCode
	a := safeStringOrEmpty(rec.Asn)
	nw := safeStringOrEmpty(rec.NetworkName)
	cn := safeStringOrEmpty(rec.CountryName)
	h := rec.Hits

	query := `
INSERT INTO usage_daily_v6
(usage_date, domain, member_name, country_code, asn, network_name, country_name, hits)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
`
	_, err := mysql.DB.Exec(query, dt, dm, mn, cc, a, nw, cn, h)
	if err != nil {
		log.Log(log.Error, "failed to upsert IPv6 usage record: %v", err)
		return err
	}
	return nil
}

// Example for retrieving IPv6 usage
func GetUsageByDomainV6(domain string, start, end time.Time) ([]mysql.UsageRecord, error) {
	// We can create a new function in mysql for v6, or replicate logic
	return getUsageByDomainV6(domain, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func getUsageByDomainV6(domain, startDate, endDate string) ([]mysql.UsageRecord, error) {
	query := `
SELECT usage_date, domain, country_code, SUM(hits) as hits
FROM usage_daily_v6
WHERE domain = ? AND usage_date BETWEEN ? AND ?
GROUP BY usage_date, domain, country_code
ORDER BY usage_date
`
	rows, err := mysql.DB.Query(query, domain, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var res []mysql.UsageRecord
	for rows.Next() {
		var r mysql.UsageRecord
		if err := rows.Scan(&r.Date, &r.Domain, &r.CountryCode, &r.Hits); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, r)
	}
	return res, nil
}

// safeStringOrEmpty converts a sql.NullString to a guaranteed non-NULL string.
func safeStringOrEmpty(ns sql.NullString) string {
	if !ns.Valid || ns.String == "" {
		return ""
	}
	return ns.String
}
