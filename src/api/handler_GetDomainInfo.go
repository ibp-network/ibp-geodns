package api

import (
	"time"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetDomainInfo(req Request) Response {
	params := req.Parameters
	log.Log(log.Debug, "handle_GetDomainInfo: QName=%s", params.QName)

	var records []DomainInfo
	currentUnixTimestamp := int(time.Now().UTC().Unix())

	TLDRecords.mu.RLock()
	defer TLDRecords.mu.RUnlock()

	for key, domain := range TLDRecords.records {
		if extractTopLevelDomain(params.QName) == domain {
			var Masters []string
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
	}

	log.Log(log.Debug, "handle_GetDomainInfo: returning %d records for qname=%s", len(records), params.QName)
	return Response{Result: records}
}
