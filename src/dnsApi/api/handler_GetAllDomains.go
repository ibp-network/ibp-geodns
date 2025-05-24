package api

import (
	log "ibp-geodns/src/common/logging"
	"time"
)

func handle_GetAllDomains(req Request) Response {
	log.Log(log.Debug, "handle_GetAllDomains called")

	currentUnixTimestamp := int(time.Now().UTC().Unix())
	records := []DomainInfo{}

	TLDRecords.mu.RLock()
	defer TLDRecords.mu.RUnlock()

	for key, domain := range TLDRecords.records {
		Masters := []string{}
		dnsPrefixes := []string{"dns-01", "dns-02", "dns-03"}

		StaticRecords.mu.RLock()
		for _, prefix := range dnsPrefixes {
			dnsName := prefix + "." + domain
			for _, dnsRecord := range StaticRecords.records {
				if dnsRecord.QName == dnsName {
					Masters = append(Masters, dnsRecord.Content)
					break
				}
			}
		}
		StaticRecords.mu.RUnlock()

		records = append(records, DomainInfo{
			DomainID:       key,
			Zone:           domain,
			Masters:        Masters,
			NotifiedSerial: currentUnixTimestamp,
			Serial:         currentUnixTimestamp,
			LastCheck:      currentUnixTimestamp,
			Kind:           "NATIVE",
		})
	}

	log.Log(log.Debug, "handle_GetAllDomains: returning %d domains", len(records))
	return Response{Result: records}
}
