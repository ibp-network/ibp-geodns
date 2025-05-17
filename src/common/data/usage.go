package data

import (
	"database/sql"
	"time"

	"ibp-geodns/src/common/data/mysql"
)

// UpsertUsage writes a usage record via the mysql package.
func UpsertUsage(date string, domain string, member sql.NullString, country string, hits int) error {
	rec := mysql.UsageRecord{
		Date:        date,
		Domain:      domain,
		MemberName:  member,
		CountryCode: country,
		Hits:        hits,
	}
	return mysql.UpsertUsageRecord(rec)
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

// ProcessDailyUsage processes stats for the given date and stores them in MySQL.
func ProcessDailyUsage(date string) {
	Stats.Mu.Lock()
	dayStats, exists := Stats.Data[date]
	Stats.Mu.Unlock()
	if !exists {
		return
	}

	for domain, ds := range dayStats {
		for country, hits := range ds.ClientStats.Countries {
			UpsertUsage(date, domain, sql.NullString{}, country, hits)
		}
		for memberName, ms := range ds.MemberStats {
			for country, hits := range ms.Countries {
				UpsertUsage(date, domain, sql.NullString{String: memberName, Valid: true}, country, hits)
			}
		}
	}
}

func startDailyUsageProcessor() {
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, time.UTC)
			time.Sleep(time.Until(next))
			y := now.AddDate(0, 0, -1).Format("2006-01-02")
			ProcessDailyUsage(y)
		}
	}()
}
