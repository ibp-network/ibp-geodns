package data

import (
	"database/sql"
	"fmt"
	"time"

	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

// usage.go for IPv4 usage
// The below functions remain for usage_daily table

type UsageRecord struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Hits        int
}

// UpsertUsageRecord inserts or updates a usage record for IPv4
func UpsertUsageRecord(rec UsageRecord) error {
	query := `
        INSERT INTO usage_daily
            (usage_date, domain, member_name, country_code, hits)
        VALUES (?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
    `
	_, err := mysql.DB.Exec(query,
		rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
	if err != nil {
		return fmt.Errorf("failed to upsert usage record: %w", err)
	}
	return nil
}

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
	Hits        int
}

func UpsertUsageRecordV6(rec UsageRecordV6) error {
	query := `
        INSERT INTO usage_daily_v6
            (usage_date, domain, member_name, country_code, hits)
        VALUES (?, ?, ?, ?, ?)
        ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
    `
	_, err := mysql.DB.Exec(query,
		rec.Date, rec.Domain, rec.MemberName, rec.CountryCode, rec.Hits)
	if err != nil {
		log.Log(log.Error, "failed to upsert IPv6 usage record: %v", err)
		return err
	}
	return nil
}

// Example for retrieving IPv6 usage
func GetUsageByDomainV6(domain string, start, end time.Time) ([]mysql.UsageRecord, error) {
	// We can create a new function in the MySQL layer or just replicate the logic:
	// For demonstration, let's assume we have a new method "GetUsageByDomainV6" in mysql.
	return getUsageByDomainV6(domain, start.Format("2006-01-02"), end.Format("2006-01-02"))
}

func getUsageByDomainV6(domain, startDate, endDate string) ([]mysql.UsageRecord, error) {
	// We'll keep the same UsageRecord struct from the "mysql" package for the row scans
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
