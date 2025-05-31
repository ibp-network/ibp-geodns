package data

import (
	"time"

	mysql "ibp-geodns/src/common/data/mysql"
)

// We make our UsageRecord an alias of mysql.UsageRecord.
// This way, "rec" is exactly the same type as "mysql.UsageRecord".
type UsageRecord = mysql.UsageRecord

// --------------------------------------------------------------------
// IPv4 usage
// --------------------------------------------------------------------

// UpsertUsageRecord stores an IPv4 usage record into usage_daily.
func UpsertUsageRecord(rec UsageRecord) error {
	return mysql.UpsertUsageRecord(rec)
}

// GetUsageByDomain fetches IPv4 usage records for a domain in [start, end].
func GetUsageByDomain(domain string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByDomain(domain, startDate, endDate)
}

// GetUsageByMember fetches IPv4 usage records for a domain+member in [start, end].
func GetUsageByMember(domain, member string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByMember(domain, member, startDate, endDate)
}

// GetUsageByCountry fetches IPv4 usage grouped by date/domain/member/country in [start, end].
func GetUsageByCountry(start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByCountry(startDate, endDate)
}

// --------------------------------------------------------------------
// IPv6 usage
// --------------------------------------------------------------------

// UpsertUsageRecordV6 stores an IPv6 usage record into usage_daily_v6.
func UpsertUsageRecordV6(rec UsageRecord) error {
	return mysql.UpsertUsageRecordV6(rec)
}

// GetUsageByDomainV6 fetches IPv6 usage records for a domain in [start, end].
func GetUsageByDomainV6(domain string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByDomainV6(domain, startDate, endDate)
}

// GetUsageByMemberV6 fetches IPv6 usage records for a domain+member in [start, end].
func GetUsageByMemberV6(domain, member string, start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByMemberV6(domain, member, startDate, endDate)
}

// GetUsageByCountryV6 fetches IPv6 usage grouped by date/domain/member/country in [start, end].
func GetUsageByCountryV6(start, end time.Time) ([]UsageRecord, error) {
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	return mysql.GetUsageByCountryV6(startDate, endDate)
}
