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

	// Detect if the client is IPv6 or IPv4
	remoteIP := net.ParseIP(params.Remote)
	isIPv6Client := false
	if remoteIP != nil && remoteIP.To4() == nil {
		isIPv6Client = true
	}

	//
	// 1) Record usage. We separate IPv4 vs. IPv6.
	//
	if isIPv6Client {
		dat.ClientHitV6(params.Remote, qname) // new function for IPv6 stats
	} else {
		dat.ClientHit(params.Remote, qname) // existing IPv4 stats
	}

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
	records = appendUniqueRecords(records, SOA)

	ACME := ProcessACME(params, id, qname)
	records = appendUniqueRecords(records, ACME)

	NS := ProcessNS(params, id, qname)
	records = appendUniqueRecords(records, NS)

	ANY := ProcessANY(params, id, qname)
	records = appendUniqueRecords(records, ANY)

	//
	// 2) Dynamic: pick IPv4 or IPv6 addresses, depending on QType.
	//
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		// Handle IPv4 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false) // false = use IPv4
	case "AAAA":
		// Handle IPv6 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true) // true = use IPv6
	case "ANY":
		// Possibly return both IPv4 + IPv6
		v4Recs, v4Member := ProcessDynamic(params, id, qname, false)
		v6Recs, v6Member := ProcessDynamic(params, id, qname, true)
		// Combine
		chosenRecords = append(v4Recs, v6Recs...)
		// If both are valid, pick whichever for "member usage" or do both
		if v6Member != "" {
			chosenMemberName = v6Member
		} else {
			chosenMemberName = v4Member
		}
	}

	if len(chosenRecords) > 0 {
		log.Log(log.Debug, "handle_DNSQuery: Found %d dynamic records", len(chosenRecords))
	}
	records = appendUniqueRecords(records, chosenRecords)

	// If we assigned a member for v4 or v6, record usage as well
	if chosenMemberName != "" {
		if isIPv6Client {
			dat.MemberHitV6(chosenMemberName, params.Remote, qname)
		} else {
			dat.MemberHit(chosenMemberName, params.Remote, qname)
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
