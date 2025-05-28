package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	"net"
)

// handle_DNSQuery processes incoming lookup requests from PowerDNS.
func handle_DNSQuery(req Request) Response {
	params := req.Parameters
	qname := strings.ToLower(strings.TrimSuffix(params.QName, "."))
	qtype := params.QType

	log.Log(log.Debug, "handle_DNSQuery: qname=%s, qtype=%s, remote=%s",
		qname, qtype, params.Remote)

	// Identify if the client is IPv6
	remoteIP := net.ParseIP(params.Remote)
	isIPv6Client := false
	if remoteIP != nil && remoteIP.To4() == nil {
		isIPv6Client = true
	}

	// Identify TLDRecords index for domain.
	var id int
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	var records []cfg.DNSRecord

	// Collect standard static records
	SOA := ProcessSOA(params, id, qname)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	records = appendUniqueRecords(records, ANY)

	// Possibly gather dynamic records (A/AAAA, or ANY).
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false)

	case "AAAA":
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true)

	case "ANY":
		// For ANY queries, we gather dynamic IPv4 + IPv6.
		v4Recs, v4Member := ProcessDynamic(params, id, qname, false)
		v6Recs, v6Member := ProcessDynamic(params, id, qname, true)
		chosenRecords = append(v4Recs, v6Recs...)

		// If multiple families found a chosen member, we only take the last found.
		if v6Member != "" {
			chosenMemberName = v6Member
		} else {
			chosenMemberName = v4Member
		}

	default:
		// For NS, SOA, CNAME, etc. no dynamic resolution beyond what's above.
		// Return existing static records (and not record usage).
		if len(records) == 0 {
			log.Log(log.Warn,
				"handle_DNSQuery: returning 0 records for qname=%s qtype=%s => NXDOMAIN or REFUSE",
				qname, qtype,
			)
			return Response{Result: []cfg.DNSRecord{}}
		}
		return Response{Result: records}
	}

	if len(chosenRecords) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d dynamic records", len(chosenRecords))
	}
	records = appendUniqueRecords(records, chosenRecords)

	// ============ USAGE RECORDING ============
	// We only record usage if a member was assigned (chosenMemberName != "").
	// That means the query actually returned a "closest" or valid member.
	//
	// This is independent of domain recognition (id != 0).
	// If you DO want to enforce recognized domain, add "&& id != 0" here:
	// if chosenMemberName != "" && id != 0 { ... }
	//
	// But per user request, we record usage for "any query that yields a non-empty chosen member."
	if chosenMemberName != "" {
		dat.RecordDnsHit(isIPv6Client, params.Remote, qname, chosenMemberName)
	}

	if len(records) == 0 {
		log.Log(log.Warn,
			"handle_DNSQuery: returning 0 records for qname=%s qtype=%s => NXDOMAIN or REFUSE",
			qname, qtype,
		)
		return Response{Result: []cfg.DNSRecord{}}
	}

	return Response{Result: records}
}
