package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"

	// ADD:
	dat "ibp-geodns/src/common/data"
)

func handle_DNSQuery(req Request) Response {
	params := req.Parameters
	qname := strings.ToLower(strings.TrimSuffix(params.QName, "."))
	qtype := params.QType

	log.Log(log.Debug, "handle_DNSQuery: qname=%s, qtype=%s, remote=%s",
		qname, qtype, params.Remote)

	// ------------------------------------------------------------------
	// 1) Record usage: we have a new DNS query from IP = params.Remote
	//    So let's call ClientHit(RemoteIP, Domain)
	// ------------------------------------------------------------------
	// If your domain is e.g. `mythos.dotters.network`, pass that as-is
	dat.ClientHit(params.Remote, qname)

	var records []cfg.DNSRecord
	var id int

	// TLDRecords usage
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
		}
	}
	TLDRecords.mu.RUnlock()

	// Gather from the different "process" steps
	SOA := ProcessSOA(params, id, qname)
	if len(SOA) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d SOA records for qname=%s", len(SOA), qname)
	}
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	if len(ACME) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d ACME records", len(ACME))
	}
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	if len(NS) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d NS records", len(NS))
	}
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	if len(ANY) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d ANY records", len(ANY))
	}
	records = appendUniqueRecords(records, ANY)

	// ------------------------------------------------------------------
	// 2) "Dynamic" might pick a single chosen member
	//    We'll modify ProcessDynamic(...) to return the chosenMemberName.
	// ------------------------------------------------------------------
	chosenRecords, chosenMemberName := ProcessDynamic(params, id, qname)
	if len(chosenRecords) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d dynamic records", len(chosenRecords))
	}
	records = appendUniqueRecords(records, chosenRecords)

	// If a chosenMember was returned from dynamic, record usage
	if chosenMemberName != "" {
		// This means we assigned the DNS request to that member
		dat.MemberHit(chosenMemberName, params.Remote, qname)
	}

	// If still no records
	if len(records) == 0 {
		// fallback check
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
			if qname == uniqueDomain {
				if qtype == "A" || qtype == "ANY" {
					log.Log(log.Warn, "DNSLookup: no dynamic record for domain %s, fallback A", qname)
					records = append(records, cfg.DNSRecord{
						DomainID: id,
						QName:    qname,
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
		log.Log(log.Warn,
			"handle_DNSQuery: returning 0 records for qname=%s qtype=%s => PDNS may REFUSE or NXDOMAIN",
			qname, qtype)
		return Response{Result: []cfg.DNSRecord{}}
	}

	// Return whatever we have
	return Response{Result: records}
}
