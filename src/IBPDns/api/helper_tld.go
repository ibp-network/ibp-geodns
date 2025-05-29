// File: src/dnsApi/api/helper_tld.go

package api

import (
	"hash/crc32"
	"strings"

	log "ibp-geodns/src/common/logging"
)

// populateTLDRecords scans through StaticRecords, extracts the eTLD+1 for each,
// and stores them in TLDRecords (with a stable ID based on CRC32).
func populateTLDRecords() {
	log.Log(log.Debug, "populateTLDRecords: starting...")

	// We'll gather them in a temporary map first.
	tmp := make(map[string]bool)

	StaticRecords.mu.RLock()
	for _, rec := range StaticRecords.records {
		zone := extractTopLevelDomain(rec.QName) // e.g. "dotters.network"
		if zone != "" {
			tmp[zone] = true
		}
	}
	StaticRecords.mu.RUnlock()

	TLDRecords.mu.Lock()
	defer TLDRecords.mu.Unlock()

	// Clear out old data
	TLDRecords.records = make(map[int]string)

	count := 0
	for domain := range tmp {
		// compute stable ID for domain
		// we mask off sign bit so it's a positive int
		id := int(crc32.ChecksumIEEE([]byte(strings.ToLower(domain)))) & 0x7FFFFFFF
		TLDRecords.records[id] = domain
		count++
		log.Log(log.Debug, "populateTLDRecords: TLDRecords[%d] = %s", id, domain)
	}

	log.Log(log.Debug, "populateTLDRecords: done, found %d unique TLD(s).", count)
}
