package api

import (
	"hash/crc32"
	"strings"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func populateTLDRecords() {
	log.Log(log.Debug, "populateTLDRecords: starting...")

	tmp := make(map[string]bool)

	StaticRecords.mu.RLock()
	for _, rec := range StaticRecords.records {
		if rec.QType != "NS" && rec.QType != "SOA" {
			continue
		}
		zone := normalizeDomain(rec.QName)
		if zone != "" {
			tmp[zone] = true
		}
	}
	StaticRecords.mu.RUnlock()

	// Include domains that exist only via dynamic services
	ServiceRecords.mu.RLock()
	for dom := range ServiceRecords.Services {
		if dom != "" {
			tmp[dom] = true
		}
	}
	ServiceRecords.mu.RUnlock()

	TLDRecords.mu.Lock()
	defer TLDRecords.mu.Unlock()

	TLDRecords.records = make(map[int]string)

	count := 0
	for domain := range tmp {
		id := int(crc32.ChecksumIEEE([]byte(strings.ToLower(domain)))) & 0x7FFFFFFF
		origID := id
		for {
			if existing, exists := TLDRecords.records[id]; !exists || strings.EqualFold(existing, domain) {
				TLDRecords.records[id] = domain
				break
			}
			// resolve collision by linear probing
			id = (id + 1) & 0x7FFFFFFF
			if id == origID {
				log.Log(log.Error, "populateTLDRecords: unable to place domain=%s due to id space exhaustion", domain)
				break
			}
		}
		count++
		log.Log(log.Debug, "populateTLDRecords: TLDRecords[%d] = %s", id, domain)
	}

	log.Log(log.Debug, "populateTLDRecords: done, found %d unique TLD(s).", count)
}
