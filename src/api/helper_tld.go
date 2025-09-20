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
		zone := extractTopLevelDomain(rec.QName)
		if zone != "" {
			tmp[zone] = true
		}
	}
	StaticRecords.mu.RUnlock()

	TLDRecords.mu.Lock()
	defer TLDRecords.mu.Unlock()

	TLDRecords.records = make(map[int]string)

	count := 0
	for domain := range tmp {
		id := int(crc32.ChecksumIEEE([]byte(strings.ToLower(domain)))) & 0x7FFFFFFF
		TLDRecords.records[id] = domain
		count++
		log.Log(log.Debug, "populateTLDRecords: TLDRecords[%d] = %s", id, domain)
	}

	log.Log(log.Debug, "populateTLDRecords: done, found %d unique TLD(s).", count)
}
