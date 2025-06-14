package data

import (
	"strings"
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

func statsEnabled() bool {
	muCacheOptions.Lock()
	defer muCacheOptions.Unlock()
	return allowStats
}

func normaliseCountryCode(code string) string {
	if len(code) != 2 {
		return "??"
	}
	return strings.ToUpper(code)
}

type dailyUsageKey struct {
	Date        string
	Domain      string
	MemberName  string
	CountryCode string
	Asn         string
	NetworkName string
	CountryName string
}

type usageMemory struct {
	mu   sync.Mutex
	data map[dailyUsageKey]int
}

var usageMem = &usageMemory{
	data: make(map[dailyUsageKey]int),
}

func RecordDnsHit(isIPv6 bool, clientIP, domain, memberName string) {
	if !statsEnabled() || domain == "" || clientIP == "" {
		return
	}

	countryCodeRaw := max.GetCountryCode(clientIP)
	countryCode := normaliseCountryCode(countryCodeRaw)

	countryName := max.GetCountryName(clientIP)
	if countryCode == "??" {
		countryName = "Unknown"
	}

	asn, netName := max.GetAsnAndNetwork(clientIP)

	if memberName == "" {
		memberName = "(none)"
	}

	now := time.Now().UTC()
	dateStr := now.Format("2006-01-02")

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
		"[RecordDnsHit] domain=%s, member=%s, ip=%s, isIPv6=%v, cc=%s => increment usageMem",
		domain, memberName, clientIP, isIPv6, countryCode)
}

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
