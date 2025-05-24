package api

import (
	"time"

	log "ibp-geodns/src/common/logging"
)

// handle_GetAllDomains returns a slice of DomainInfo for each TLDRecord
func handle_GetAllDomains(req Request) Response {
	log.Log(log.Debug, "handle_GetAllDomains: called by remote=%s", req.Parameters.Remote)

	currentUnixTimestamp := int(time.Now().UTC().Unix())
	records := []DomainInfo{}

	// For debugging: show how many TLDRecords we have
	TLDRecords.mu.RLock()
	countTLD := len(TLDRecords.records)
	log.Log(log.Debug, "handle_GetAllDomains: TLDRecords has %d entries", countTLD)
	// optional: dump them
	for key, domain := range TLDRecords.records {
		log.Log(log.Debug, "handle_GetAllDomains: TLDRecord[%d] = %s", key, domain)
	}
	defer TLDRecords.mu.RUnlock()

	// Build a DomainInfo for each TLD
	for key, domain := range TLDRecords.records {
		masters := []string{}

		dnsPrefixes := []string{"dns-01", "dns-02", "dns-03"}

		StaticRecords.mu.RLock()
		// Gather each prefix from the static records as "masters"
		for _, prefix := range dnsPrefixes {
			dnsName := prefix + "." + domain
			for _, dnsRecord := range StaticRecords.records {
				if dnsRecord.QName == dnsName {
					masters = append(masters, dnsRecord.Content)
					break
				}
			}
		}
		StaticRecords.mu.RUnlock()

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
		log.Log(log.Debug, "handle_GetAllDomains: record[%d] => DomainID=%d, Zone=%s, Masters=%v",
			i, di.DomainID, di.Zone, di.Masters)
	}

	return Response{Result: records}
}
