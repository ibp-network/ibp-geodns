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

	log.Log(log.Debug, "handle_DNSQuery: qname=%s, qtype=%s, remote=%s", qname, qtype, params.Remote)

	// Check if this is IPv6 or IPv4
	remoteIP := net.ParseIP(params.Remote)
	isIPv6Client := false
	if remoteIP != nil && remoteIP.To4() == nil {
		isIPv6Client = true
	}

	// Identify the domain in TLDRecords (id != 0 means recognized domain).
	var id int
	TLDRecords.mu.RLock()
	for key, tld := range TLDRecords.records {
		if extractTopLevelDomain(qname) == strings.ToLower(tld) {
			id = key
			break
		}
	}
	TLDRecords.mu.RUnlock()

	// --- Gather static records: SOA, ACME, NS, ANY ---
	var records []cfg.DNSRecord

	SOA := ProcessSOA(params, id, qname)
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	records = appendUniqueRecords(records, ANY)

	// --- Possibly gather dynamic (A/AAAA) records ---
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false)
	case "AAAA":
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true)
	case "ANY":
		// For ANY queries, we provide both A + AAAA if available.
		v4Recs, v4Member := ProcessDynamic(params, id, qname, false)
		v6Recs, v6Member := ProcessDynamic(params, id, qname, true)
		chosenRecords = append(v4Recs, v6Recs...)

		// If both members are valid, we pick whichever is found last
		// purely for logging. The usage counters can reflect both.
		if v6Member != "" {
			chosenMemberName = v6Member
		} else {
			chosenMemberName = v4Member
		}
		// for all other qtypes, no dynamic resolution beyond the above
	}

	if len(chosenRecords) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d dynamic records", len(chosenRecords))
	}
	records = appendUniqueRecords(records, chosenRecords)

	// === USAGE RECORDING LOGIC ===
	// We only record usage if domain is recognized (id != 0) AND qtype is A/AAAA/ANY.
	if id != 0 && (qtype == "A" || qtype == "AAAA" || qtype == "ANY") {
		// 1) Always record a domain-level usage (with memberName="").
		dat.RecordDnsHit(isIPv6Client, params.Remote, qname, "")

		// 2) If a dynamic resolution chose a specific member, also record that usage.
		if chosenMemberName != "" {
			dat.RecordDnsHit(isIPv6Client, params.Remote, qname, chosenMemberName)
		}
	}

	// If no records at all, we return NXDOMAIN or REFUSE.
	if len(records) == 0 {
		log.Log(log.Warn,
			"handle_DNSQuery: returning 0 records for qname=%s qtype=%s => NXDOMAIN or REFUSE",
			qname, qtype,
		)
		return Response{Result: []cfg.DNSRecord{}}
	}

	return Response{Result: records}
}
