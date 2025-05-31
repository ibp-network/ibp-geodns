package data

import (
	"database/sql"
	cfg "ibp-geodns/src/common/config"
	mysql "ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
	"time"
)

// InitOptions allows selective initialization of data subsystems.
type InitOptions struct {
	UseLocalOfficialCaches bool // if true, load/save local+official results
	UseUsageStats          bool // if true, track usage daily stats
}

// Init selectively initializes data subsystems based on InitOptions.
func Init(opts InitOptions) {
	log.Log(log.Debug, "[data.Init] Starting with options: %+v", opts)

	// Always initialize MySQL (for events, usage records, etc).
	go mysql.Init()

	// (1) Set the global flags for saving caches:
	SetCacheOptions(opts.UseLocalOfficialCaches, opts.UseUsageStats)

	// Initialize the global Stats struct (IPv4)
	Stats = &StatMap{
		Data: make(map[string]map[string]*DailyStats),
	}

	// Initialize the global Stats6 struct (IPv6)
	Stats6 = &StatMap{
		Data: make(map[string]map[string]*DailyStats),
	}

	// If we have local/official caching or usage stats, load them now, then save them.
	if opts.UseLocalOfficialCaches || opts.UseUsageStats {
		LoadAllCaches()
		SaveAllCaches()
		// auto-save official & local caches
		go startAutoUpdate()
	}

	// If usage is needed, start daily usage processing
	if opts.UseUsageStats {
		go startDailyUsageProcessor()
	}
}

// MemberEnable sets Override=false on a member and records an event.
func MemberEnable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not enable member; does not exist")
		return
	}
	member.Override = false
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil)
}

// MemberDisable sets Override=true on a member and records an event.
func MemberDisable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not disable member; does not exist")
		return
	}
	member.Override = true
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberDisable", name, "", "", false, "Member has enabled override.", nil)
}

// IsMemberOnlineForDomain checks official results for IPv4.
func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// site-level
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// domain-level
	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// endpoint-level
	for _, er := range endpoints {
		if er.Domain == domain {
			for _, r := range er.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	return true
}

// IsMemberOnlineForDomainIPv6 checks official results for IPv6.
func IsMemberOnlineForDomainIPv6(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// site-level (only if IsIPv6 == true)
	for _, sr := range sites {
		if !sr.IsIPv6 {
			continue
		}
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// domain-level (only if IsIPv6 == true)
	for _, dr := range domains {
		if !dr.IsIPv6 {
			continue
		}
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// endpoint-level (only if IsIPv6 == true)
	for _, er := range endpoints {
		if !er.IsIPv6 {
			continue
		}
		if er.Domain == domain {
			for _, r := range er.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	return true
}

// startAutoUpdate periodically calls SaveAllCaches() so we keep disk caches updated.
func startAutoUpdate() {
	ticker := time.NewTicker(90 * time.Second)
	go func() {
		for range ticker.C {
			SaveAllCaches()
		}
	}()
}

// startDailyUsageProcessor processes daily usage at 00:05 UTC each day.
func startDailyUsageProcessor() {
	go func() {
		for {
			now := time.Now().UTC()
			// Wait until next 00:05 UTC
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 5, 0, 0, time.UTC)
			time.Sleep(time.Until(next))

			// We process usage for "yesterday"
			y := now.AddDate(0, 0, -1).Format("2006-01-02")
			log.Log(log.Info, "startDailyUsageProcessor: processing daily usage for %s", y)

			ProcessDailyUsage(y)
		}
	}()
}

// ProcessDailyUsage handles both IPv4 and IPv6 stats for the given date.
func ProcessDailyUsage(date string) {
	processDailyUsageV4(date)
	processDailyUsageV6(date)
}

// processDailyUsageV4 writes IPv4 stats from memory to usage_daily in DB.
func processDailyUsageV4(date string) {
	Stats.Mu.Lock()
	dailyMap, ok := Stats.Data[date]
	if !ok {
		log.Log(log.Info, "processDailyUsageV4: no IPv4 usage found for %s", date)
		Stats.Mu.Unlock()
		return
	}

	log.Log(log.Info, "processDailyUsageV4: found IPv4 usage entries for date %s", date)

	for domain, dailyStats := range dailyMap {
		// Overall usage: no specific member => (none)
		for countryCode, hits := range dailyStats.ClientStats.Countries {
			rec := UsageRecord{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{Valid: false},
				CountryCode: countryCode,
				Hits:        hits,
			}
			err := UpsertUsageRecord(rec)
			if err != nil {
				log.Log(log.Error, "processDailyUsageV4 upsert error: %v", err)
			}
		}
		// Per-member usage
		for memberName, ms := range dailyStats.MemberStats {
			for countryCode, hits := range ms.Countries {
				rec := UsageRecord{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{Valid: true, String: memberName},
					CountryCode: countryCode,
					Hits:        hits,
				}
				err := UpsertUsageRecord(rec)
				if err != nil {
					log.Log(log.Error, "processDailyUsageV4 upsert error: %v", err)
				}
			}
		}
	}

	// Remove processed data from memory
	delete(Stats.Data, date)
	Stats.Mu.Unlock()

	log.Log(log.Info, "processDailyUsageV4: done writing IPv4 usage for %s", date)
}

// processDailyUsageV6 writes IPv6 stats from memory to usage_daily_v6 in DB.
func processDailyUsageV6(date string) {
	Stats6.Mu.Lock()
	dailyMap, ok := Stats6.Data[date]
	if !ok {
		log.Log(log.Info, "processDailyUsageV6: no IPv6 usage found for %s", date)
		Stats6.Mu.Unlock()
		return
	}

	log.Log(log.Info, "processDailyUsageV6: found IPv6 usage entries for date %s", date)

	for domain, dailyStats := range dailyMap {
		// Overall usage: no specific member => (none)
		for countryCode, hits := range dailyStats.ClientStats.Countries {
			rec := UsageRecord{
				Date:        date,
				Domain:      domain,
				MemberName:  sql.NullString{Valid: false},
				CountryCode: countryCode,
				Hits:        hits,
			}
			err := UpsertUsageRecordV6(rec)
			if err != nil {
				log.Log(log.Error, "processDailyUsageV6 upsert error: %v", err)
			}
		}
		// Per-member usage
		for memberName, ms := range dailyStats.MemberStats {
			for countryCode, hits := range ms.Countries {
				rec := UsageRecord{
					Date:        date,
					Domain:      domain,
					MemberName:  sql.NullString{Valid: true, String: memberName},
					CountryCode: countryCode,
					Hits:        hits,
				}
				err := UpsertUsageRecordV6(rec)
				if err != nil {
					log.Log(log.Error, "processDailyUsageV6 upsert error: %v", err)
				}
			}
		}
	}

	// Remove processed data from memory
	delete(Stats6.Data, date)
	Stats6.Mu.Unlock()

	log.Log(log.Info, "processDailyUsageV6: done writing IPv6 usage for %s", date)
}
