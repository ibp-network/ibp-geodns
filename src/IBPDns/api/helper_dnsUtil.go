package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"

	"golang.org/x/net/publicsuffix"
)

func appendUniqueRecords(records []cfg.DNSRecord, newRecords []cfg.DNSRecord) []cfg.DNSRecord {
	for _, newRecord := range newRecords {
		if !containsRecord(records, newRecord) {
			records = append(records, newRecord)
		}
	}
	return records
}

func containsRecord(records []cfg.DNSRecord, record cfg.DNSRecord) bool {
	for _, r := range records {
		if r.QName == record.QName && r.QType == record.QType && r.Content == record.Content {
			return true
		}
	}
	return false
}

func extractTopLevelDomain(domain string) string {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "wss://")
	domain = strings.TrimPrefix(domain, "ws://")
	if idx := strings.Index(domain, "/"); idx != -1 {
		domain = domain[:idx]
	}
	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return ""
	}
	return eTLDPlusOne
}
