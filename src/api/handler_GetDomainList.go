package api

import (
	"fmt"
	"strings"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetDomainList(req Request) Response {
	params := req.Parameters
	log.Log(log.Debug, "handle_GetDomainList: zonename=%s, domain_id=%s", params.Zonename, params.DomainID)

	var records []cfg.DNSRecord
	id, _ := lookupTLDID(params.Zonename)
	zone := normalizeDomain(params.Zonename)
	ServiceRecords.mu.RLock()
	_, hasDynamicZone := ServiceRecords.Services[zone]
	ServiceRecords.mu.RUnlock()

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, record := range StaticRecords.records {
		if zone == normalizeDomain(record.QName) {
			records = append(records, cfg.DNSRecord{
				DomainID: id,
				QName:    normalizeDomain(record.QName),
				QType:    record.QType,
				Content:  record.Content,
				TTL:      record.TTL,
				Auth:     true,
			})
		}
	}

	if hasDynamicZone {
		records = appendMissingDynamicZoneAuthorityRecords(records, id, zone)
	}

	log.Log(log.Debug, "handle_GetDomainList: returning %d records for zonename=%s", len(records), params.Zonename)
	return Response{Result: records}
}

func appendMissingDynamicZoneAuthorityRecords(records []cfg.DNSRecord, id int, zone string) []cfg.DNSRecord {
	hasSOA := false
	hasNS := false
	for _, record := range records {
		if !strings.EqualFold(normalizeDomain(record.QName), zone) {
			continue
		}

		switch {
		case strings.EqualFold(record.QType, "SOA"):
			hasSOA = true
		case strings.EqualFold(record.QType, "NS"):
			hasNS = true
		}
	}

	if !hasSOA {
		records = append(records, cfg.DNSRecord{
			DomainID: id,
			QName:    zone,
			QType:    "SOA",
			Content:  fmt.Sprintf("dns-01.%s. hostmaster.%s. %d 3600 600 1209600 3600", zone, zone, int(time.Now().UTC().Unix())),
			TTL:      3600,
			Auth:     true,
		})
	}
	if !hasNS {
		records = append(records, cfg.DNSRecord{
			DomainID: id,
			QName:    zone,
			QType:    "NS",
			Content:  fmt.Sprintf("dns-01.%s.", zone),
			TTL:      3600,
			Auth:     true,
		})
	}

	return records
}
