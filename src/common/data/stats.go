package data

import (
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

// dailyUsageKey is the unique combination that we store in memory before flushing to DB.
type dailyUsageKey struct {
	Date        string
	Domain      string
	MemberName  string
	CountryCode string
	Asn         string
	NetworkName string
	CountryName string
}

// usageMemory holds a map of dailyUsageKey -> hits
type usageMemory struct {
	mu   sync.Mutex
	data map[dailyUsageKey]int
}

// global in-memory usage stats
var usageMem = &usageMemory{
	data: make(map[dailyUsageKey]int),
}

// RecordDnsHit is called from the DNS query logic with client IP, domain, and assigned member.
func RecordDnsHit(isIPv6 bool, clientIP, domain, memberName string) {
	if domain == "" || clientIP == "" {
		return
	}
	// Do geo lookups
	countryCode := max.GetCountryCode(clientIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	countryName := max.GetCountryName(clientIP)
	asn, netName := max.GetAsnAndNetwork(clientIP)
	if memberName == "" {
		memberName = "(none)"
	}

	// build key
	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02") // example: "2025-05-28"

	key := dailyUsageKey{
		Date:        dateStr,
		Domain:      domain,
		MemberName:  memberName,
		CountryCode: countryCode,
		Asn:         asn,
		NetworkName: netName,
		CountryName: countryName,
	}

	usageMem.mu.Lock()
	usageMem.data[key]++
	usageMem.mu.Unlock()

	log.Log(log.Debug, "[RecordDnsHit] domain=%s, member=%s, ip=%s, isIPv6=%v => increment usageMem", domain, memberName, clientIP, isIPv6)
}

// FlushUsageToDatabase processes all usage for a specific date from memory and writes it to MySQL.
func FlushUsageToDatabase(date string) {
	usageMem.mu.Lock()
	defer usageMem.mu.Unlock()

	// gather all keys for the given date
	var keysToFlush []dailyUsageKey
	for k := range usageMem.data {
		if k.Date == date {
			keysToFlush = append(keysToFlush, k)
		}
	}

	if len(keysToFlush) == 0 {
		log.Log(log.Info, "FlushUsageToDatabase: no usage found for date=%s", date)
		return
	}

	log.Log(log.Info, "FlushUsageToDatabase: found %d usage entries for date=%s", len(keysToFlush), date)

	for _, k := range keysToFlush {
		hits := usageMem.data[k]

		rec := UsageRecord{
			Date:        k.Date,
			Domain:      k.Domain,
			MemberName:  k.MemberName,
			CountryCode: k.CountryCode,
			Asn:         k.Asn,
			NetworkName: k.NetworkName,
			CountryName: k.CountryName,
			Hits:        hits,
		}

		err := UpsertUsageRecord(rec)
		if err != nil {
			log.Log(log.Error, "FlushUsageToDatabase: upsert error for domain=%s member=%s: %v", k.Domain, k.MemberName, err)
		}

		// remove it from memory
		delete(usageMem.data, k)
	}

	log.Log(log.Info, "FlushUsageToDatabase: done flushing %d usage records for date=%s", len(keysToFlush), date)
}
