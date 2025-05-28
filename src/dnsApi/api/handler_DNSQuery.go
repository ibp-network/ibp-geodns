package api

import (
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	"net"
)

// handle_DNSQuery processes incoming lookup requests from PowerDNS
func handle_DNSQuery(req Request) Response {
	params := req.Parameters
	qname := strings.ToLower(strings.TrimSuffix(params.QName, "."))
	qtype := params.QType

	log.Log(log.Debug, "handle_DNSQuery: qname=%s, qtype=%s, remote=%s", qname, qtype, params.Remote)

	// Determine if client is IPv6
	remoteIP := net.ParseIP(params.Remote)
	isIPv6Client := false
	if remoteIP != nil && remoteIP.To4() == nil {
		isIPv6Client = true
	}

	// Look up TLDRecords
	var id int
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	// Gather static/dynamic records as before
	var records []cfg.DNSRecord

	SOA := ProcessSOA(params, id, qname)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	records = appendUniqueRecords(records, ANY)

	// Determine if we should do dynamic lookups (A/AAAA only).
	// If it's anything else, we skip dynamic resolution and skip usage increments.
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		// If domain recognized, pick IPv4
		if id != 0 {
			chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false)
		}
	case "AAAA":
		// If domain recognized, pick IPv6
		if id != 0 {
			chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true)
		}
	default:
		// For anything else (NS, ANY, etc.), do not record usage or do dynamic resolution.
		log.Log(log.Debug, "handle_DNSQuery: qtype=%s not A/AAAA, skipping usage increments", qtype)
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

	// Only record usage if qtype is A/AAAA AND domain recognized (id != 0).
	// We'll do it after we know if a member was chosen or not.
	if (qtype == "A" || qtype == "AAAA") && id != 0 {
		// First increment usage with memberName = "" (none).
		// This indicates a domain-level request.
		dat.RecordDnsHit(isIPv6Client, params.Remote, qname, "")

		// If a member was chosen, we update usage under that member as well.
		if chosenMemberName != "" {
			dat.RecordDnsHit(isIPv6Client, params.Remote, qname, chosenMemberName)
		}
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
