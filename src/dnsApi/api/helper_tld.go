// File: src/dnsApi/api/helper_tld.go

package api

import (
	log "ibp-geodns/src/common/logging"
)

// populateTLDRecords scans through StaticRecords, extracts the eTLD+1 for each,
// and stores them in TLDRecords (with auto-increment integer keys).
func populateTLDRecords() {
	log.Log(log.Debug, "populateTLDRecords: starting...")

	// We'll gather them in a temporary map first.
	// Keys are the domain name, value is just a bool placeholder.
	tlds := make(map[string]bool)

	StaticRecords.mu.RLock()
	for _, rec := range StaticRecords.records {
		zone := extractTopLevelDomain(rec.QName) // e.g. "dotters.network"
		if zone != "" {
			tlds[zone] = true
		}
	}
	StaticRecords.mu.RUnlock()

	// Now write them into TLDRecords
	TLDRecords.mu.Lock()
	// Clear out any old data
	TLDRecords.records = make(map[int]string)

	count := 0
	for z := range tlds {
		count++
		TLDRecords.records[count] = z
		log.Log(log.Debug, "populateTLDRecords: TLDRecords[%d] = %s", count, z)
	}
	TLDRecords.mu.Unlock()

	log.Log(log.Debug, "populateTLDRecords: done, found %d unique TLD(s).", count)
}
