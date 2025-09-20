package api

import (
	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetDomainList(req Request) Response {
	params := req.Parameters
	log.Log(log.Debug, "handle_GetDomainList: zonename=%s, domain_id=%s", params.Zonename, params.DomainID)

	var records []cfg.DNSRecord
	var id int

	TLDRecords.mu.RLock()
	for key, domain := range TLDRecords.records {
		if extractTopLevelDomain(params.Zonename) == domain {
			id = key
		}
	}
	TLDRecords.mu.RUnlock()

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, record := range StaticRecords.records {
		if extractTopLevelDomain(params.Zonename) == extractTopLevelDomain(record.QName) {
			records = append(records, cfg.DNSRecord{
				DomainID: id,
				QName:    record.QName,
				QType:    record.QType,
				Content:  record.Content,
				TTL:      record.TTL,
				Auth:     true,
			})
		}
	}

	log.Log(log.Debug, "handle_GetDomainList: returning %d records for zonename=%s", len(records), params.Zonename)
	return Response{Result: records}
}
