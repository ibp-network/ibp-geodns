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

	// Identify TLDRecords index (id)
	var id int
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	// We'll gather all standard static records:
	var records []cfg.DNSRecord

	SOA := ProcessSOA(params, id, qname)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	records = appendUniqueRecords(records, ANY)

	// For dynamic resolution, we handle A, AAAA, ANY:
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		// IPv4 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false)

	case "AAAA":
		// IPv6 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true)

	case "ANY":
		// We still gather dynamic A/AAAA for ANY queries if the domain is recognized.
		v4Recs, v4Member := ProcessDynamic(params, id, qname, false)
		v6Recs, v6Member := ProcessDynamic(params, id, qname, true)
		chosenRecords = append(v4Recs, v6Recs...)

		// For usage, though, we will skip increments for ANY (see below).
		// But for completeness, we pick a "chosen" member to reflect which address was used last.
		if v6Member != "" {
			chosenMemberName = v6Member
		} else {
			chosenMemberName = v4Member
		}

	default:
		// For NS, SOA, CNAME, etc.: We do NOT do dynamic resolution beyond the above
		// (though some existing code in ANY handles it).
		// We do not record usage for anything except A/AAAA,
		// but we DO want to return any static/dynamic records that already exist in 'records'.
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

	// ==========================
	// USAGE RECORDING SECTION
	// ==========================
	// Only record usage stats if:
	// 1) The query is A or AAAA
	// 2) We recognized the domain (id != 0)
	if (qtype == "A" || qtype == "AAAA") && id != 0 {
		// First increment usage with memberName=""
		dat.RecordDnsHit(isIPv6Client, params.Remote, qname, "")

		// If a specific member was chosen, record usage for that member as well
		if chosenMemberName != "" {
			dat.RecordDnsHit(isIPv6Client, params.Remote, qname, chosenMemberName)
		}
	}
	// if qtype == ANY (or others), we do nothing for usage

	if len(records) == 0 {
		log.Log(log.Warn,
			"handle_DNSQuery: returning 0 records for qname=%s qtype=%s => NXDOMAIN or REFUSE",
			qname, qtype,
		)
		return Response{Result: []cfg.DNSRecord{}}
	}

	return Response{Result: records}
}
