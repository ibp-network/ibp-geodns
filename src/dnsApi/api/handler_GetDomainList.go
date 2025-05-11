package api

import (
	cfg "ibp-geodns/src/common/config"
	"net/http"
)

func DnsQuery_List(w http.ResponseWriter, r *http.Request, req Request) Response {
	var records []cfg.DNSRecord
	var id int

	for key, domain := range TLDRecords.records {
		if extractTopLevelDomain(req.Parameters.Zonename) == domain {
			id = key
		}
	}

	for _, record := range StaticRecords.records {
		if extractTopLevelDomain(req.Parameters.Zonename) == extractTopLevelDomain(record.QName) {
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

	return Response{Result: records}
}
