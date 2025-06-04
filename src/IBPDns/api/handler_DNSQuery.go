package api

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"strings"
)

// handle_DNSQuery processes "lookup" requests from PowerDNS.
// We do not base it on the client IP for IPv6/IPv4 decisions. Instead, we see QType: "A", "AAAA", or "ANY".
// Then we produce records accordingly.
func handle_DNSQuery(req Request) Response {
	params := req.Parameters
	qname := strings.ToLower(strings.TrimSuffix(params.QName, "."))
	qtype := params.QType

	log.Log(log.Debug, "handle_DNSQuery: qname=%s, qtype=%s, remote=%s", qname, qtype, params.Remote)

	// Find TLDRecords index for domain
	var id int
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	var finalRecords []cfg.DNSRecord

	// Always gather the static records relevant for e.g. SOA, NS, ACME, etc.
	SOA := ProcessSOA(params, id, qname)
	finalRecords = appendUniqueRecords(finalRecords, SOA)

	ACME := ProcessACME(params, id, qname)
	finalRecords = appendUniqueRecords(finalRecords, ACME)

	NS := ProcessNS(params, id, qname)
	finalRecords = appendUniqueRecords(finalRecords, NS)

	ANY := ProcessANY(params, id, qname)
	if qtype == "ANY" && len(ANY) > 0 {
		// We'll keep these, except we'll also do dynamic IPv4/IPv6 below
		finalRecords = appendUniqueRecords(finalRecords, ANY)
	}

	// Next handle dynamic A, AAAA, or ANY
	switch qtype {
	case "A":
		// Return only IPv4 dynamic records
		records4, chosen4 := ProcessDynamic(params, id, qname, false)
		if len(records4) > 0 {
			finalRecords = appendUniqueRecords(finalRecords, records4)
			log.Log(log.Debug, "handle_DNSQuery(A): chose member=%s, records=%d", chosen4, len(records4))
		}
	case "AAAA":
		// Return only IPv6 dynamic records
		records6, chosen6 := ProcessDynamic(params, id, qname, true)
		if len(records6) > 0 {
			finalRecords = appendUniqueRecords(finalRecords, records6)
			log.Log(log.Debug, "handle_DNSQuery(AAAA): chose member=%s, records=%d", chosen6, len(records6))
		}
	case "ANY":
		// gather both IPv4 + IPv6
		r4, c4 := ProcessDynamic(params, id, qname, false)
		r6, c6 := ProcessDynamic(params, id, qname, true)
		if len(r4) > 0 {
			finalRecords = appendUniqueRecords(finalRecords, r4)
			log.Log(log.Debug, "handle_DNSQuery(ANY): IPv4 from member=%s, recs=%d", c4, len(r4))
		}
		if len(r6) > 0 {
			finalRecords = appendUniqueRecords(finalRecords, r6)
			log.Log(log.Debug, "handle_DNSQuery(ANY): IPv6 from member=%s, recs=%d", c6, len(r6))
		}
	default:
		// handled above for static
	}

	if len(finalRecords) == 0 {
		log.Log(log.Debug, "handle_DNSQuery: returning 0 records => NXDOMAIN or REFUSE for q=%s", qname)
	}

	return Response{Result: finalRecords}
}
