package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

func handle_DNSQuery(req Request) Response {
	var records []cfg.DNSRecord
	c := cfg.GetConfig()
	var id int

	domain := strings.ToLower(strings.TrimSuffix(req.Parameters.QName, "."))

	// TLDRecords usage
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(domain) == strings.ToLower(tld) {
			id = key
		}
	}
	TLDRecords.mu.RUnlock()

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

	if len(records) == 0 {
		var uniqueDomains []string
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
					log.Log(log.Warn, "DNSLookup: no dynamic record for domain %s, fallback A", domain)
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
		return Response{Result: []cfg.DNSRecord{}}
	}
	return Response{Result: records}
}
