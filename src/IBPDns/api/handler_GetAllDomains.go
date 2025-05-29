package api

import (
	"time"

	log "ibp-geodns/src/common/logging"
)

// handle_GetAllDomains returns a slice of DomainInfo for each TLDRecord
func handle_GetAllDomains(req Request) Response {
	log.Log(log.Debug, "handle_GetAllDomains: called by remote=%s", req.Parameters.Remote)

	// Capture the current timestamp for use in all DomainInfo structs
	currentUnixTimestamp := int(time.Now().UTC().Unix())

	// Copy the TLDRecords to avoid locking TLDRecords while also locking StaticRecords
	TLDRecords.mu.RLock()
	tldRecordsCopy := make(map[int]string, len(TLDRecords.records))
	for k, v := range TLDRecords.records {
		tldRecordsCopy[k] = v
	}
	TLDRecords.mu.RUnlock()

	// Debug: Log how many we copied
	log.Log(log.Debug, "handle_GetAllDomains: TLDRecords has %d entries", len(tldRecordsCopy))
	for key, domain := range tldRecordsCopy {
		log.Log(log.Debug, "handle_GetAllDomains: TLDRecord[%d] = %s", key, domain)
	}

	// Build a DomainInfo for each TLD
	var records []DomainInfo
	for key, domain := range tldRecordsCopy {
		// Gather any NS "masters" (i.e., dns-01, dns-02, etc.) from the static records
		masters := gatherMasters(domain)

		records = append(records, DomainInfo{
			DomainID:       key,
			Zone:           domain,
			Masters:        masters,
			NotifiedSerial: currentUnixTimestamp,
			Serial:         currentUnixTimestamp,
			LastCheck:      currentUnixTimestamp,
			Kind:           "NATIVE",
		})
	}

	log.Log(log.Debug, "handle_GetAllDomains: returning %d DomainInfo records", len(records))
	// Optionally, show them all
	for i, di := range records {
		log.Log(
			log.Debug,
			"handle_GetAllDomains: record[%d] => DomainID=%d, Zone=%s, Masters=%v",
			i, di.DomainID, di.Zone, di.Masters,
		)
	}

	return Response{Result: records}
}

// gatherMasters scans StaticRecords for each dns-XX subdomain.
func gatherMasters(domain string) []string {
	dnsPrefixes := []string{"dns-01", "dns-02", "dns-03"}
	var masters []string

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, prefix := range dnsPrefixes {
		dnsName := prefix + "." + domain
		for _, dnsRecord := range StaticRecords.records {
			if dnsRecord.QName == dnsName {
				masters = append(masters, dnsRecord.Content)
				break
			}
		}
	}
	return masters
}
