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

	// Merge old "client" + "member" counters into one usage increment:
	dat.RecordDnsHit(isIPv6Client, params.Remote, qname, "")

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

	// 2) Dynamic: pick IPv4 or IPv6 addresses, depending on QType
	var chosenRecords []cfg.DNSRecord
	var chosenMemberName string

	switch qtype {
	case "A":
		// Handle IPv4 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, false)
	case "AAAA":
		// Handle IPv6 dynamic
		chosenRecords, chosenMemberName = ProcessDynamic(params, id, qname, true)
	case "ANY":
		// Possibly return both IPv4 + IPv6
		v4Recs, v4Member := ProcessDynamic(params, id, qname, false)
		v6Recs, v6Member := ProcessDynamic(params, id, qname, true)
		chosenRecords = append(v4Recs, v6Recs...)

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

	// If we assigned a member for v4 or v6, store that in usage counters
	if chosenMemberName != "" {
		// Overwrite the previously empty "memberName" usage with the chosen member now
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
