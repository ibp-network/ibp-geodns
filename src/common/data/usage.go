package data

import (
	"database/sql"
	"time"

	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

// UsageRecordDetailed is a new structure to capture everything
// If you are storing it in usage_daily, you will need new columns for
// asn, network_name, and country_name.
type UsageRecordDetailed struct {
	Date        string
	Domain      string
	MemberName  sql.NullString
	CountryCode string
	Asn         sql.NullString
	NetworkName sql.NullString
	CountryName sql.NullString
	Hits        int
}

// UpsertUsageDetailed writes or updates a row in usage_daily with new columns
func UpsertUsageDetailed(rec UsageRecordDetailed) error {
	// Adjust column names as needed. Example schema:
	// usage_date DATE, domain VARCHAR(255), member_name VARCHAR(255),
	// country_code VARCHAR(8), asn VARCHAR(64), network_name VARCHAR(128), country_name VARCHAR(128), hits INT
	// with (usage_date, domain, member_name, country_code, asn, network_name, country_name) as unique key or partial key

	query := `
	    INSERT INTO usage_daily (
            usage_date,
            domain,
            member_name,
            country_code,
            asn,
            network_name,
            country_name,
            hits
        ) 
	        VALUES (?,?,?,?,?,?,?,?) 
	        ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)
	`
	_, err := mysql.DB.Exec(query,
		rec.Date,
		rec.Domain,
		rec.MemberName,
		rec.CountryCode,
		rec.Asn,
		rec.NetworkName,
		rec.CountryName,
		rec.Hits,
	)
	if err != nil {
		log.Log(log.Error, "failed to upsert usage record detailed: %v", err)
		return err
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

// ProcessDailyUsage processes stats for the given date and stores them in MySQL.
// We show an example of also storing the newly added fields (asn, networkName, countryName).
func ProcessDailyUsage(date string) {
	Stats.Mu.Lock()
	dayStats, exists := Stats.Data[date]
	Stats.Mu.Unlock()
	if !exists {
		return
	}

	for domain, ds := range dayStats {
		// 1) Overall "ClientStats" aggregates
		for countryCode, hits := range ds.ClientStats.Countries {
			rec := UsageRecordDetailed{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{}, // no member
				CountryCode: countryCode,
				Asn:         sql.NullString{String: "", Valid: false},
				NetworkName: sql.NullString{String: "", Valid: false},
				CountryName: sql.NullString{String: "", Valid: false},
				Hits:        hits,
			}
			UpsertUsageDetailed(rec)
		}

		// 1a) Also store ASNs from ds.ClientStats
		for asnVal, hits := range ds.ClientStats.Asns {
			rec := UsageRecordDetailed{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{}, // no member
				CountryCode: "",               // no country code
				Asn:         sql.NullString{String: asnVal, Valid: asnVal != ""},
				NetworkName: sql.NullString{String: "", Valid: false},
				CountryName: sql.NullString{String: "", Valid: false},
				Hits:        hits,
			}
			UpsertUsageDetailed(rec)
		}

		// 1b) Also store Networks from ds.ClientStats
		for netVal, hits := range ds.ClientStats.Networks {
			rec := UsageRecordDetailed{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{},
				CountryCode: "",
				Asn:         sql.NullString{String: "", Valid: false},
				NetworkName: sql.NullString{String: netVal, Valid: netVal != ""},
				CountryName: sql.NullString{String: "", Valid: false},
				Hits:        hits,
			}
			UpsertUsageDetailed(rec)
		}

		// 1c) Also store CountryName from ds.ClientStats
		for cnVal, hits := range ds.ClientStats.CountryNames {
			rec := UsageRecordDetailed{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{},
				CountryCode: "", // we do store iso code separately, but here is the "full name"
				Asn:         sql.NullString{String: "", Valid: false},
				NetworkName: sql.NullString{String: "", Valid: false},
				CountryName: sql.NullString{String: cnVal, Valid: cnVal != ""},
				Hits:        hits,
			}
			UpsertUsageDetailed(rec)
		}

		// 2) MemberStats breakdown
		for memberName, ms := range ds.MemberStats {
			for countryCode, hits := range ms.Countries {
				rec := UsageRecordDetailed{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{String: memberName, Valid: true},
					CountryCode: countryCode,
					Asn:         sql.NullString{String: "", Valid: false},
					NetworkName: sql.NullString{String: "", Valid: false},
					CountryName: sql.NullString{String: "", Valid: false},
					Hits:        hits,
				}
				UpsertUsageDetailed(rec)
			}

			// Similarly store ASNs
			for asnVal, hits := range ms.Asns {
				rec := UsageRecordDetailed{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{String: memberName, Valid: true},
					CountryCode: "",
					Asn:         sql.NullString{String: asnVal, Valid: asnVal != ""},
					NetworkName: sql.NullString{String: "", Valid: false},
					CountryName: sql.NullString{String: "", Valid: false},
					Hits:        hits,
				}
				UpsertUsageDetailed(rec)
			}

			// Networks
			for netVal, hits := range ms.Networks {
				rec := UsageRecordDetailed{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{String: memberName, Valid: true},
					CountryCode: "",
					Asn:         sql.NullString{String: "", Valid: false},
					NetworkName: sql.NullString{String: netVal, Valid: netVal != ""},
					CountryName: sql.NullString{String: "", Valid: false},
					Hits:        hits,
				}
				UpsertUsageDetailed(rec)
			}

			// CountryName
			for cnVal, hits := range ms.CountryNames {
				rec := UsageRecordDetailed{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{String: memberName, Valid: true},
					CountryCode: "",
					Asn:         sql.NullString{String: "", Valid: false},
					NetworkName: sql.NullString{String: "", Valid: false},
					CountryName: sql.NullString{String: cnVal, Valid: cnVal != ""},
					Hits:        hits,
				}
				UpsertUsageDetailed(rec)
			}
		}
	}
}

// ------------------------------------------------------------------------
// Add the missing startDailyUsageProcessor() to fix "undefined" reference
// ------------------------------------------------------------------------
func startDailyUsageProcessor() {
	go func() {
		for {
			now := time.Now().UTC()
			// We'll run the daily usage processing at (00:05 UTC) each day
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, time.UTC)
			time.Sleep(time.Until(next))

			// Process "yesterday's" usage
			y := now.AddDate(0, 0, -1).Format("2006-01-02")
			log.Log(log.Info, "startDailyUsageProcessor: processing daily usage for %s", y)
			ProcessDailyUsage(y)
		}
	}()
}
