package data

import (
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

/* -----------------------------------------------------------------------
   Internal helpers
   ---------------------------------------------------------------------*/

// statsEnabled returns the current allowStats flag (protected by the
// same mutex used in cache.go).  We keep it in a small helper to avoid
// directly touching the package‑scope variables from multiple files.
func statsEnabled() bool {
	muCacheOptions.Lock()
	defer muCacheOptions.Unlock()
	return allowStats
}

/* -----------------------------------------------------------------------
   In‑memory structures
   ---------------------------------------------------------------------*/

// dailyUsageKey is the unique combination that we store in memory
// before flushing to DB.
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

// global in‑memory usage stats
var usageMem = &usageMemory{
	data: make(map[dailyUsageKey]int),
}

/* -----------------------------------------------------------------------
   Recording
   ---------------------------------------------------------------------*/

// RecordDnsHit is called from the DNS query logic with client IP,
// domain, and assigned member.
func RecordDnsHit(isIPv6 bool, clientIP, domain, memberName string) {
	if !statsEnabled() || domain == "" || clientIP == "" {
		return
	}

	// Geo lookups
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
	dateStr := now.Format("2006-01-02") // e.g. "2025-05-28"

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

	log.Log(log.Debug,
		"[RecordDnsHit] domain=%s, member=%s, ip=%s, isIPv6=%v => increment usageMem",
		domain, memberName, clientIP, isIPv6)
}

/* -----------------------------------------------------------------------
   Flushing
   ---------------------------------------------------------------------*/

// FlushUsageToDatabase writes *all* accumulated usage (for every date)
// to MySQL and clears the in‑memory map.
//
// If usage tracking is disabled (`allowStats == false`) the function exits
// immediately and does nothing.
//
// NOTE: Previously this function only flushed a single date, which left
// stale keys in RAM indefinitely. The implementation now iterates over
// **every** key ensuring the map cannot grow without bound.
func FlushUsageToDatabase(triggerDate string) {
	if !statsEnabled() {
		return
	}

	usageMem.mu.Lock()
	defer usageMem.mu.Unlock()

	if len(usageMem.data) == 0 {
		log.Log(log.Info,
			"[FlushUsageToDatabase] No usage to flush (triggerDate=%s)",
			triggerDate)
		return
	}

	log.Log(log.Info,
		"[FlushUsageToDatabase] Flushing %d usage records (triggerDate=%s)",
		len(usageMem.data), triggerDate)

	flushed := 0
	for k, hits := range usageMem.data {
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

		if err := UpsertUsageRecord(rec); err != nil {
			log.Log(log.Error,
				"[FlushUsageToDatabase] upsert error domain=%s member=%s date=%s: %v",
				rec.Domain, rec.MemberName, rec.Date, err)
			// continue even if one record fails
			continue
		}

		// remove the key after successful flush
		delete(usageMem.data, k)
		flushed++
	}

	log.Log(log.Info,
		"[FlushUsageToDatabase] Completed flush: %d records written, map size now %d",
		flushed, len(usageMem.data))
}
