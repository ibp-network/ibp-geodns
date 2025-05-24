package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

func handle_DNSQuery(req Request) Response {
	// Basic logging for each lookup
	qName := strings.ToLower(strings.TrimSuffix(req.Parameters.QName, "."))
	qType := req.Parameters.QType
	remoteIP := req.Parameters.Remote
	log.Log(log.Info, "handle_DNSQuery: qName=%s, qType=%s, remote=%s", qName, qType, remoteIP)

	var records []cfg.DNSRecord
	var id int

	// TLD lookup
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qName) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	// Process each potential record set
	SOA := ProcessSOA(req.Parameters, id, qName)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(req.Parameters, id, qName)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(req.Parameters, id, qName)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(req.Parameters, id, qName)
	records = appendUniqueRecords(records, ANY)

	Dynamic := ProcessDynamic(req.Parameters, id, qName)
	records = appendUniqueRecords(records, Dynamic)

	// If we still have no records, fallback logic
	if len(records) == 0 {
		var uniqueDomains []string
		c := cfg.GetConfig()
		for _, service := range c.Services {
			for _, provider := range service.Providers {
				for _, url := range provider.RpcUrls {
					u := max.ParseUrl(url)
					uniqueDomains = append(uniqueDomains, u.Domain)
				}
			}
		}

		for _, uniqueDomain := range uniqueDomains {
			if qName == uniqueDomain {
				if qType == "A" || qType == "ANY" {
					log.Log(log.Warn, "DNSLookup: no dynamic record for domain %s, fallback A", qName)
					records = append(records, cfg.DNSRecord{
						DomainID: id,
						QName:    qName,
						QType:    "A",
						Content:  "192.96.202.175",
						TTL:      30,
						Auth:     true,
					})
				}
			}
		}
	}

	if len(records) == 0 {
		log.Log(log.Info, "DNSLookup: returning empty record set for qName=%s qType=%s", qName, qType)
		return Response{Result: []cfg.DNSRecord{}}
	}

	// Log final record set for debugging
	log.Log(log.Info, "DNSLookup: returning %d records for qName=%s qType=%s", len(records), qName, qType)
	return Response{Result: records}
}
