package api

import (
	cfg "github.com/ibp-network/ibp-geodns-libs/config"
)

func findZone(name string) (int, string, bool) {
	normalizedName := normalizeDomain(name)
	if normalizedName == "" {
		return 0, "", false
	}

	id, ok := lookupTLDID(normalizedName)
	if !ok {
		return 0, "", false
	}

	return id, normalizedName, true
}

func domainMetadataForZone(domain string) map[string][]string {
	if _, _, ok := findZone(domain); !ok {
		return map[string][]string{}
	}

	return map[string][]string{
		"PRESIGNED": []string{"0"},
	}
}

func recordForResponse(record cfg.DNSRecord, id int, domain string) cfg.DNSRecord {
	return cfg.DNSRecord{
		DomainID: id,
		QName:    normalizeDomain(domain),
		QType:    record.QType,
		Content:  record.Content,
		TTL:      record.TTL,
		Auth:     true,
	}
}
