package api

import (
	"time"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetAllDomains(req Request) Response {
	log.Log(log.Debug, "handle_GetAllDomains: called by remote=%s", req.Parameters.Remote)

	currentUnixTimestamp := int(time.Now().UTC().Unix())

	TLDRecords.mu.RLock()
	tldRecordsCopy := make(map[int]string, len(TLDRecords.records))
	for k, v := range TLDRecords.records {
		tldRecordsCopy[k] = v
	}
	TLDRecords.mu.RUnlock()

	log.Log(log.Debug, "handle_GetAllDomains: TLDRecords has %d entries", len(tldRecordsCopy))
	for key, domain := range tldRecordsCopy {
		log.Log(log.Debug, "handle_GetAllDomains: TLDRecord[%d] = %s", key, domain)
	}

	var records []DomainInfo
	for key, domain := range tldRecordsCopy {
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
	for i, di := range records {
		log.Log(
			log.Debug,
			"handle_GetAllDomains: record[%d] => DomainID=%d, Zone=%s, Masters=%v",
			i, di.DomainID, di.Zone, di.Masters,
		)
	}

	return Response{Result: records}
}

func gatherMasters(domain string) []string {
	dnsPrefixes := []string{"dns-01", "dns-02", "dns-03"}
	var masters []string

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, prefix := range dnsPrefixes {
		dnsName := prefix + "." + domain
		for _, dnsRecord := range StaticRecords.records {
			if normalizeDomain(dnsRecord.QName) == normalizeDomain(dnsName) {
				masters = append(masters, dnsRecord.Content)
				break
			}
		}
	}
	return masters
}
