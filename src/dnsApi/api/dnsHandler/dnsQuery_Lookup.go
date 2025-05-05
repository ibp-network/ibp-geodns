package api

import (
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"net/http"
	"strings"
)

// dnsQuery_Lookup handles DNS lookup queries based on the provided parameters.
func DnsQuery_Lookup(w http.ResponseWriter, r *http.Request, req Request) Response {
	var records []cfg.DNSRecord

	c := cfg.GetConfig()
	var id int

	domain := strings.ToLower(strings.TrimSuffix(req.Parameters.QName, "."))

	// Store Client Stats
	go dat.ClientHit(req.Parameters.Remote, domain)

	// Initiate TopLevelDomains mutex read lock
	TLDRecords.mu.RLock()
	defer TLDRecords.mu.RUnlock()

	// Fetch domain id based on domain's index in array.
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(domain) == strings.ToLower(tld) {
			id = key
		}
	}

	// Collect records, ensuring no duplicates
	SOA := ProcessSOA(req.Parameters, id, domain)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(req.Parameters, id, domain)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(req.Parameters, id, domain)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(req.Parameters, id, domain)
	records = appendUniqueRecords(records, ANY)

	Dynamic := ProcessDynamic(req.Parameters, id, domain)
	records = appendUniqueRecords(records, Dynamic)

	// Default record if requested domain is valid
	if len(records) == 0 {
		var uniqueDomains []string
		uniqueDomains = make([]string, 0)

		for _, service := range c.Services {
			for _, provider := range service.Providers {
				for _, url := range provider.RpcUrls {
					u := max.ParseUrl(url)
					uniqueDomains = append(uniqueDomains, u.Domain)
				}
			}
		}

		for _, uniqueDomain := range uniqueDomains {
			if domain == uniqueDomain {
				if req.Parameters.QType == "A" || req.Parameters.QType == "ANY" {
					log.Log(log.Info, "DNSLookup: No records found for domain %s, returning default result", domain)
					records = append(records, cfg.DNSRecord{
						DomainID: id,
						QName:    domain,
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
		// We need to return an empty record so the client knows there is no result
		return Response{Result: []cfg.DNSRecord{}}
	} else {
		// Return compiled records
		return Response{Result: records}
	}
}

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
